package daemon

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/daemon/ipc"
	"github.com/po1o/prompto/src/log"
	"github.com/po1o/prompto/src/prompt"
	"github.com/po1o/prompto/src/runtime"
	pathRuntime "github.com/po1o/prompto/src/runtime/path"

	"google.golang.org/grpc"
)

// Server is the gRPC adapter. It owns the wire (listener, gRPC server, the
// proto handlers, and per-stream cancel tracking for gRPC handlers) and
// delegates all business state — render orchestration, toggles, config
// watching, the reload worker, the device cache — to its embedded Daemon.
type Server struct {
	ipc.UnimplementedDaemonServiceServer
	listener   net.Listener
	done       chan struct{}
	lockFile   *LockFile
	grpcServer *grpc.Server
	// core owns render/session state, lifecycle, watchers, toggles, cache.
	core *Daemon
	// primaryStreams maps a session ID to the gRPC stream cancel func, so a
	// new render request for the same session can force the prior wire
	// handler to return immediately rather than wait for the next poll.
	primaryStreams map[string]primaryStreamState
	streamMu       sync.Mutex
	shutdownOnce   sync.Once
}

type primaryStreamState struct {
	cancel    context.CancelFunc
	requestID string
}

func NewServer(configPath string) (*Server, error) {
	resolvedPath := resolveServerConfigPath(configPath)

	lockFile, err := NewLockFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	server := &Server{
		lockFile:       lockFile,
		done:           make(chan struct{}),
		primaryStreams: make(map[string]primaryStreamState),
	}
	server.core = NewFromConfigWithDeviceCache(resolvedPath, nil, nil)
	server.core.SetOnStop(server.Stop)

	return server, nil
}

func (server *Server) Start() error {
	listener, err := ipc.Listen()
	if err != nil {
		_ = server.lockFile.Release()
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	ipc.RegisterDaemonServiceServer(grpcServer, server)

	server.listener = listener
	server.grpcServer = grpcServer

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChannel
		server.Stop()
	}()

	return grpcServer.Serve(listener)
}

func (server *Server) Done() <-chan struct{} {
	return server.done
}

func (server *Server) Stop() {
	server.shutdownOnce.Do(func() {
		// StopSilently tears down daemon-owned watchers and the reload worker.
		server.core.StopSilently()

		if server.grpcServer != nil {
			server.grpcServer.GracefulStop()
		}

		_ = log.SetOutputPath("")

		_ = ipc.CleanupSocket()
		_ = server.lockFile.Release()
		close(server.done)
	})
}

func (server *Server) RenderPrompt(
	request *ipc.PromptRequest,
	stream grpc.ServerStreamingServer[ipc.PromptResponse],
) error {
	// Apply any already-queued config reload before starting a new render so the
	// first prompt after save reflects updated config.
	server.core.ProcessPendingConfigReload()
	// Also check config mtime in case render starts before fsnotify event delivery.
	server.core.ReloadIfConfigFileUpdated()

	if request.Version != ipc.ProtocolVersion {
		return fmt.Errorf("protocol version mismatch: client=%d server=%d", request.Version, ipc.ProtocolVersion)
	}

	var flags *runtime.Flags
	if request.Flags != nil {
		flags = ipc.ProtoToFlags(request.Flags)
	}
	if flags == nil {
		flags = &runtime.Flags{}
	}

	if flags.ConfigPath == "" {
		flags.ConfigPath = server.core.ConfigPath()
	}

	sessionID := resolveServerSessionID(request.Pid, request.SessionId)

	flags.SegmentToggles = server.core.SessionToggles(sessionID)

	streamContext := stream.Context()
	releaseStream := func() {}
	if flags.Type == "" || flags.Type == prompt.PRIMARY {
		var cancel context.CancelFunc
		streamContext, cancel = context.WithCancel(stream.Context())
		releaseStream = server.replacePrimaryStream(sessionID, request.RequestId, cancel)
		defer func() {
			cancel()
			releaseStream()
		}()
	}

	initial := server.core.StartRender(RenderRequest{
		SessionID: sessionID,
		Flags:     flags,
		Cancel:    CancelKindForRepaint(request.Repaint),
	})

	if initial.Type == "stopped" {
		return fmt.Errorf("daemon is stopped")
	}

	lastBundle := initial.Bundle
	sequence := initial.Sequence

	// One-shot prompt types (tooltip/valid/error/debug/etc.) should return the
	// computed bundle directly and not enter streaming update flow.
	if flags.Type != "" && flags.Type != prompt.PRIMARY {
		return stream.Send(makePromptResponse("complete", request.RequestId, &lastBundle))
	}

	if err := streamContext.Err(); err != nil {
		return nil
	}

	if err := stream.Send(makePromptResponse("update", request.RequestId, &initial.Bundle)); err != nil {
		return err
	}

	for {
		update, ok := server.core.NextUpdate(streamContext, sessionID, sequence)
		if !ok {
			break
		}

		sequence = update.Sequence
		if update.Segment == renderCompletePayload {
			lastBundle = update.Bundle
			break
		}

		lastBundle = update.Bundle
		if err := streamContext.Err(); err != nil {
			return nil
		}
		if err := stream.Send(makePromptResponse("update", request.RequestId, &update.Bundle)); err != nil {
			return err
		}
	}

	if err := streamContext.Err(); err != nil {
		return nil
	}

	return stream.Send(makePromptResponse("complete", request.RequestId, &lastBundle))
}

func (server *Server) ToggleSegment(
	_ context.Context,
	request *ipc.ToggleSegmentRequest,
) (*ipc.ToggleSegmentResponse, error) {
	sessionID := resolveServerSessionID(0, request.SessionId)
	server.core.ToggleSegment(sessionID, request.Segments)

	return &ipc.ToggleSegmentResponse{Success: true}, nil
}

func (server *Server) CacheClear(_ context.Context, _ *ipc.CacheClearRequest) (*ipc.CacheClearResponse, error) {
	server.core.DeviceCache().Clear()
	server.core.ResetToggles()

	cache.DeleteAll(cache.Device)
	cache.DeleteAll(cache.Session)
	return &ipc.CacheClearResponse{Success: true}, nil
}

func (server *Server) CacheSetTTL(_ context.Context, request *ipc.CacheSetTTLRequest) (*ipc.CacheSetTTLResponse, error) {
	if request.Days <= 0 {
		return &ipc.CacheSetTTLResponse{Success: false}, nil
	}

	ttl := time.Duration(request.Days) * 24 * time.Hour
	server.core.DeviceCache().SetDefaultTTL(ttl)
	cache.Set(cache.Device, cache.TTL, int(request.Days), cache.INFINITE)
	return &ipc.CacheSetTTLResponse{Success: true}, nil
}

func (server *Server) CacheGetTTL(_ context.Context, _ *ipc.CacheGetTTLRequest) (*ipc.CacheGetTTLResponse, error) {
	if ttlDays, ok := cache.Get[int](cache.Device, cache.TTL); ok && ttlDays > 0 {
		return &ipc.CacheGetTTLResponse{Days: int32(ttlDays)}, nil
	}

	defaultDays := int(server.core.DeviceCache().GetDefaultTTL() / (24 * time.Hour))
	if defaultDays <= 0 {
		defaultDays = 7
	}

	return &ipc.CacheGetTTLResponse{Days: int32(defaultDays)}, nil
}

func (server *Server) SetLogging(_ context.Context, request *ipc.SetLoggingRequest) (*ipc.SetLoggingResponse, error) {
	if request.Path == "" {
		return loggingResponse(log.SetOutputPath(""))
	}

	log.Enable(true)
	if err := log.SetOutputPath(request.Path); err != nil {
		return loggingResponse(err)
	}
	log.Debug("daemon logging to file")

	return &ipc.SetLoggingResponse{Success: true}, nil
}

func resolveServerConfigPath(configPath string) string {
	if configPath == "" {
		return filepath.Clean(defaultServerConfigPath())
	}

	resolved := pathRuntime.ReplaceTildePrefixWithHomeDir(configPath)
	absolutePath, err := filepath.Abs(resolved)
	if err != nil {
		return filepath.Clean(resolved)
	}

	return filepath.Clean(absolutePath)
}

func defaultServerConfigPath() string {
	return config.DefaultPath()
}

func makePromptResponse(responseType, requestID string, bundle *PromptBundle) *ipc.PromptResponse {
	if bundle == nil {
		bundle = &PromptBundle{}
	}

	prompts := map[string]*ipc.Prompt{
		"primary": {Text: bundle.Primary},
		"right":   {Text: bundle.RPrompt},
	}

	if bundle.Secondary != "" {
		prompts["secondary"] = &ipc.Prompt{Text: bundle.Secondary}
	}

	if bundle.Transient != "" {
		prompts["transient"] = &ipc.Prompt{Text: bundle.Transient}
	}

	if bundle.RTransient != "" {
		prompts["rtransient"] = &ipc.Prompt{Text: bundle.RTransient}
	}

	for name, text := range bundle.Extras {
		prompts[name] = &ipc.Prompt{Text: text}
	}

	return &ipc.PromptResponse{
		Type:      responseType,
		RequestId: requestID,
		Prompts:   prompts,
	}
}

func (server *Server) replacePrimaryStream(sessionID, requestID string, cancel context.CancelFunc) func() {
	server.streamMu.Lock()
	previous, ok := server.primaryStreams[sessionID]
	server.primaryStreams[sessionID] = primaryStreamState{
		requestID: requestID,
		cancel:    cancel,
	}
	server.streamMu.Unlock()

	if ok && previous.cancel != nil && previous.requestID != requestID {
		previous.cancel()
	}

	return func() {
		server.streamMu.Lock()
		defer server.streamMu.Unlock()

		current, ok := server.primaryStreams[sessionID]
		if !ok || current.requestID != requestID {
			return
		}

		delete(server.primaryStreams, sessionID)
	}
}

func resolveServerSessionID(pid int32, sessionID string) string {
	if pid > 0 {
		return fmt.Sprint(pid)
	}

	if sessionID == "" {
		return "default"
	}

	return sessionID
}

func loggingResponse(err error) (*ipc.SetLoggingResponse, error) {
	if err != nil {
		return &ipc.SetLoggingResponse{Success: false, Error: err.Error()}, nil
	}

	return &ipc.SetLoggingResponse{Success: true}, nil
}
