package daemon

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jandedobbeleer/oh-my-posh/src/cache"
	"github.com/jandedobbeleer/oh-my-posh/src/config"
	"github.com/jandedobbeleer/oh-my-posh/src/daemon/ipc"
	"github.com/jandedobbeleer/oh-my-posh/src/log"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime"
	pathRuntime "github.com/jandedobbeleer/oh-my-posh/src/runtime/path"

	"google.golang.org/grpc"
)

const defaultConfigReloadPollInterval = 250 * time.Millisecond

type Server struct {
	ipc.UnimplementedDaemonServiceServer
	core           *Daemon
	configPath     string
	lockFile       *LockFile
	listener       net.Listener
	grpcServer     *grpc.Server
	done           chan struct{}
	shutdownOnce   sync.Once
	configCache    *ConfigCache
	configWatcher  *ConfigWatcher
	binaryWatcher  *BinaryWatcher
	deviceCache    *DeviceCache
	segmentToggles map[string]map[string]bool
	toggleMu       sync.RWMutex
}

func NewServer(configPath string) (*Server, error) {
	resolvedPath := resolveServerConfigPath(configPath)

	lockFile, err := NewLockFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	server := &Server{
		core:           nil,
		configPath:     resolvedPath,
		lockFile:       lockFile,
		done:           make(chan struct{}),
		deviceCache:    NewDeviceCache(),
		segmentToggles: make(map[string]map[string]bool),
	}
	server.core = NewFromConfigWithDeviceCache(resolvedPath, nil, server.deviceCache)

	configCache := NewConfigCache()
	configWatcher, err := NewConfigWatcher(configCache)
	if err == nil {
		server.configCache = configCache
		server.configWatcher = configWatcher
		server.refreshConfigWatches()
		go server.configReloadLoop()
	}

	binaryPath, err := os.Executable()
	if err == nil {
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

	err = grpcServer.Serve(listener)
	if err != nil {
		return err
	}

	return nil
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

	sessionID := request.SessionId
	if request.Pid > 0 {
		sessionID = fmt.Sprint(request.Pid)
	}

	if sessionID == "" {
		sessionID = "default"
	}

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
	sessionID := request.SessionId
	if sessionID == "" {
		sessionID = "default"
	}

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
		err := log.SetOutputPath("")
		if err != nil {
			return &ipc.SetLoggingResponse{Success: false, Error: err.Error()}, nil
		}

		return &ipc.SetLoggingResponse{Success: true}, nil
	}

	log.Enable(true)
	if err := log.SetOutputPath(request.Path); err != nil {
		return &ipc.SetLoggingResponse{Success: false, Error: err.Error()}, nil
	}
	log.Debug("daemon logging to file")

	return &ipc.SetLoggingResponse{Success: true}, nil
}

func (server *Server) configReloadLoop() {
	missing := false
	ticker := time.NewTicker(defaultConfigReloadPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-server.done:
			return
		case <-ticker.C:
			if server.configCache == nil || server.configPath == "" {
				continue
			}

			_, ok := server.configCache.Get(server.configPath)
			if ok {
				missing = false
				continue
			}

			if missing {
				continue
			}

			missing = true
			server.core.Reload(func() {
				cache.Set(cache.Device, config.RELOAD, true, cache.INFINITE)
				server.core.Reset()
			})

			if server.refreshConfigWatches() {
				missing = false
			}
		}
	}
}

func (server *Server) refreshConfigWatches() bool {
	if server.configWatcher == nil || server.configCache == nil || server.configPath == "" {
		return false
	}

	cfg, err := config.Parse(server.configPath)
	if err != nil {
		return false
	}

	server.configCache.Set(server.configPath, cfg, cfg.FilePaths)
	return server.configWatcher.Watch(server.configPath, cfg.FilePaths) == nil
}

func resolveServerConfigPath(configPath string) string {
	if configPath == "" {
		return ""
	}

	if strings.HasPrefix(configPath, "https://") || strings.HasPrefix(configPath, "http://") {
		return configPath
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
	for key, value := range source {
		cloned[key] = value
	}

	return cloned
}
