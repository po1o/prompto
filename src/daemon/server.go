package daemon

import (
	"context"
	"fmt"
	"maps"
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
	"github.com/po1o/prompto/src/runtime"
	pathRuntime "github.com/po1o/prompto/src/runtime/path"

	"google.golang.org/grpc"
)

type Server struct {
	ipc.UnimplementedDaemonServiceServer
	listener   net.Listener
	done       chan struct{}
	lockFile   *LockFile
	grpcServer *grpc.Server
	// core owns render/session state and idle lifecycle.
	core          *Daemon
	configWatcher *ConfigWatcher
	binaryWatcher *BinaryWatcher
	deviceCache   *DeviceCache
	// configReloadCh is a coalescing signal channel (buffer=1).
	configReloadCh chan struct{}
	// segmentToggles keeps per-session runtime toggle state.
	segmentToggles map[string]map[string]bool
	configPath     string
	toggleMu       sync.RWMutex
	shutdownOnce   sync.Once
}

func NewServer(configPath string) (*Server, error) {
	resolvedPath := resolveServerConfigPath(configPath)

	lockFile, err := NewLockFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	server := &Server{
		configPath:     resolvedPath,
		lockFile:       lockFile,
		done:           make(chan struct{}),
		deviceCache:    NewDeviceCache(),
		configReloadCh: make(chan struct{}, 1),
		segmentToggles: make(map[string]map[string]bool),
	}
	server.core = NewFromConfigWithDeviceCache(resolvedPath, nil, server.deviceCache)

	configWatcher, err := NewConfigWatcher(server.requestConfigReload)
	if err == nil {
		server.configWatcher = configWatcher
		server.refreshConfigWatches()
		go server.configReloadWorker()
	}

	binaryPath, err := os.Executable()
	if err == nil {
		// If executable is replaced while running, stop; client auto-start path will launch new one.
		watcher, watchErr := NewBinaryWatcher(binaryPath, func() {
			server.Stop()
		})
		if watchErr == nil {
			server.binaryWatcher = watcher
		}
	}

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
		server.core.Stop()

		if server.grpcServer != nil {
			server.grpcServer.GracefulStop()
		}

		if server.configWatcher != nil {
			_ = server.configWatcher.Close()
		}

		if server.binaryWatcher != nil {
			_ = server.binaryWatcher.Close()
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
	if request.Version != ipc.ProtocolVersion {
		return fmt.Errorf("protocol version mismatch: client=%d server=%d", request.Version, ipc.ProtocolVersion)
	}

	flags := ipc.ProtoToFlags(request.Flags)
	if flags == nil {
		flags = &runtime.Flags{}
	}

	if flags.ConfigPath == "" {
		flags.ConfigPath = server.configPath
	}

	sessionID := resolveServerSessionID(request.Pid, request.SessionId)

	flags.SegmentToggles = server.sessionToggles(sessionID)

	initial := server.core.StartRender(RenderRequest{
		SessionID: sessionID,
		Flags:     flags,
		Repaint:   request.Repaint,
	})

	if initial.Type == "stopped" {
		return fmt.Errorf("daemon is stopped")
	}

	lastBundle := initial.Bundle
	sequence := initial.Sequence

	if err := stream.Send(makePromptResponse("update", request.RequestId, initial.Bundle)); err != nil {
		return err
	}

	for {
		update, ok := server.core.NextUpdate(stream.Context(), sessionID, sequence)
		if !ok {
			break
		}

		sequence = update.Sequence
		if update.Segment == renderCompletePayload {
			break
		}

		lastBundle = update.Bundle
		if err := stream.Send(makePromptResponse("update", request.RequestId, update.Bundle)); err != nil {
			return err
		}
	}

	return stream.Send(makePromptResponse("complete", request.RequestId, lastBundle))
}

func (server *Server) ToggleSegment(
	_ context.Context,
	request *ipc.ToggleSegmentRequest,
) (*ipc.ToggleSegmentResponse, error) {
	sessionID := resolveServerSessionID(0, request.SessionId)

	currentToggleSet := server.sessionToggles(sessionID)

	for _, segment := range request.Segments {
		if currentToggleSet[segment] {
			delete(currentToggleSet, segment)
			continue
		}

		currentToggleSet[segment] = true
	}

	server.toggleMu.Lock()
	server.segmentToggles[sessionID] = currentToggleSet
	server.toggleMu.Unlock()

	return &ipc.ToggleSegmentResponse{Success: true}, nil
}

func (server *Server) CacheClear(_ context.Context, _ *ipc.CacheClearRequest) (*ipc.CacheClearResponse, error) {
	server.deviceCache.Clear()

	server.toggleMu.Lock()
	server.segmentToggles = make(map[string]map[string]bool)
	server.toggleMu.Unlock()

	cache.DeleteAll(cache.Device)
	cache.DeleteAll(cache.Session)
	return &ipc.CacheClearResponse{Success: true}, nil
}

func (server *Server) CacheSetTTL(_ context.Context, request *ipc.CacheSetTTLRequest) (*ipc.CacheSetTTLResponse, error) {
	if request.Days <= 0 {
		return &ipc.CacheSetTTLResponse{Success: false}, nil
	}

	ttl := time.Duration(request.Days) * 24 * time.Hour
	server.deviceCache.SetDefaultTTL(ttl)
	cache.Set(cache.Device, cache.TTL, int(request.Days), cache.INFINITE)
	return &ipc.CacheSetTTLResponse{Success: true}, nil
}

func (server *Server) CacheGetTTL(_ context.Context, _ *ipc.CacheGetTTLRequest) (*ipc.CacheGetTTLResponse, error) {
	if ttlDays, ok := cache.Get[int](cache.Device, cache.TTL); ok && ttlDays > 0 {
		return &ipc.CacheGetTTLResponse{Days: int32(ttlDays)}, nil
	}

	defaultDays := int(server.deviceCache.GetDefaultTTL() / (24 * time.Hour))
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

func (server *Server) configReloadWorker() {
	// Single consumer for config reload requests.
	// Reload itself is guarded by ReloadGate (inside server.core.Reload),
	// so running it from one worker keeps sequencing easy to reason about.
	for {
		select {
		case <-server.done:
			return
		case <-server.configReloadCh:
			if server.configPath == "" {
				continue
			}

			server.core.Reload(func() {
				cache.Set(cache.Device, config.RELOAD, true, cache.INFINITE)
				server.core.Reset()
			})

			server.refreshConfigWatches()
		}
	}
}

func (server *Server) requestConfigReload(configPath string) {
	// Ignore unrelated watched files. We only reload for this daemon instance's root config.
	if configPath == "" || configPath != server.configPath {
		return
	}

	select {
	case <-server.done:
		return
	default:
	}

	select {
	// Buffered channel of size 1 coalesces bursts of fsnotify events.
	// If a reload is already queued/in progress, extra signals are redundant.
	case server.configReloadCh <- struct{}{}:
	default:
	}
}

func (server *Server) refreshConfigWatches() {
	if server.configWatcher == nil || server.configPath == "" {
		return
	}

	cfg, err := config.Parse(server.configPath)
	if err != nil {
		return
	}

	// Re-register all resolved files (root + extends + symlink targets).
	// Watch() is idempotent for already tracked files/dirs.
	_ = server.configWatcher.Watch(server.configPath, cfg.FilePaths)
}

func resolveServerConfigPath(configPath string) string {
	if configPath == "" {
		return ""
	}

	resolved := pathRuntime.ReplaceTildePrefixWithHomeDir(configPath)
	absolutePath, err := filepath.Abs(resolved)
	if err != nil {
		return filepath.Clean(resolved)
	}

	return filepath.Clean(absolutePath)
}

func makePromptResponse(responseType, requestID string, bundle PromptBundle) *ipc.PromptResponse {
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

	for name, text := range bundle.Extras {
		prompts[name] = &ipc.Prompt{Text: text}
	}

	return &ipc.PromptResponse{
		Type:      responseType,
		RequestId: requestID,
		Prompts:   prompts,
	}
}

func (server *Server) sessionToggles(sessionID string) map[string]bool {
	server.toggleMu.RLock()
	existing, ok := server.segmentToggles[sessionID]
	server.toggleMu.RUnlock()
	if ok {
		return cloneToggleMap(existing)
	}

	baseToggles, _ := cache.Get[map[string]bool](cache.Session, cache.TOGGLECACHE)
	cloned := cloneToggleMap(baseToggles)

	server.toggleMu.Lock()
	server.segmentToggles[sessionID] = cloned
	server.toggleMu.Unlock()

	return cloneToggleMap(cloned)
}

func cloneToggleMap(source map[string]bool) map[string]bool {
	if len(source) == 0 {
		return map[string]bool{}
	}

	cloned := make(map[string]bool, len(source))
	maps.Copy(cloned, source)

	return cloned
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
