package daemon

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/prompt"
)

type Daemon struct {
	// service owns render lifecycle and stream updates for all sessions.
	service *Service
	// sessions tracks live shell PIDs so idle shutdown is based on process exits, not RPC churn.
	sessions *SessionManager
	// deviceCache is shared across sessions/renders and survives per-session engine resets.
	deviceCache *DeviceCache
	onStop      func()
	// idleTimeout is armed when there are no tracked sessions.
	idleTimeout time.Duration
	// idleToken invalidates stale timers when activity resumes.
	idleToken uint64
	stopped   atomic.Bool
	mu        sync.Mutex
}

func New(renderer promptBundleRenderer) *Daemon {
	return NewWithIdleTimeoutAndDeviceCache(5*time.Minute, renderer, nil)
}

func NewFromConfig(configPath string, renderer promptBundleRenderer) *Daemon {
	cfg := config.Load(configPath)
	return NewWithIdleTimeoutAndDeviceCache(cfg.GetDaemonIdleTimeout(), renderer, nil)
}

func NewWithIdleTimeout(idleTimeout time.Duration, renderer promptBundleRenderer) *Daemon {
	return NewWithIdleTimeoutAndDeviceCache(idleTimeout, renderer, nil)
}

func NewFromConfigWithDeviceCache(configPath string, renderer promptBundleRenderer, deviceCache *DeviceCache) *Daemon {
	cfg := config.Load(configPath)
	return NewWithIdleTimeoutAndDeviceCache(cfg.GetDaemonIdleTimeout(), renderer, deviceCache)
}

func NewWithIdleTimeoutAndDeviceCache(idleTimeout time.Duration, renderer promptBundleRenderer, deviceCache *DeviceCache) *Daemon {
	if deviceCache == nil {
		deviceCache = NewDeviceCache()
	}

	registry := NewEngineRegistry(prompt.New)
	gate := NewReloadGate()
	service := NewService(registry, gate, renderer)
	service.pipeline.deviceCache = newPromptDeviceCacheBridge(deviceCache)

	daemon := &Daemon{
		service:     service,
		deviceCache: deviceCache,
		idleTimeout: idleTimeout,
	}
	daemon.sessions = NewSessionManager(daemon.onSessionUnregister, daemon.onAllSessionsEnded)

	// Start the idle timer immediately; it is canceled on first tracked render.
	daemon.mu.Lock()
	daemon.scheduleIdleStopLocked()
	daemon.mu.Unlock()

	return daemon
}

func (daemon *Daemon) DeviceCache() *DeviceCache {
	return daemon.deviceCache
}

func (daemon *Daemon) StartRender(request RenderRequest) RenderResponse {
	if daemon.stopped.Load() {
		return RenderResponse{Type: "stopped"}
	}

	// Any tracked PID render is considered activity and cancels pending idle stop.
	daemon.registerSessionPID(request)

	return daemon.service.StartRender(request)
}

func (daemon *Daemon) NextUpdate(ctx context.Context, sessionID string, after uint64) (RenderResponse, bool) {
	if daemon.stopped.Load() {
		return RenderResponse{}, false
	}

	return daemon.service.NextUpdate(ctx, sessionID, after)
}

func (daemon *Daemon) CompleteSession(sessionID string) {
	if daemon.stopped.Load() {
		return
	}

	daemon.service.CompleteSession(sessionID)

	pid, ok := parseSessionPID(sessionID)
	if ok {
		// PID-backed sessions are lifecycle-managed by SessionManager callbacks.
		daemon.sessions.Unregister(pid)
		return
	}

	daemon.scheduleIdleIfNoSessions()
}

func (daemon *Daemon) Reload(action func()) {
	if daemon.stopped.Load() {
		return
	}

	daemon.service.Reload(action)
}

func (daemon *Daemon) Snapshot() (active int, reloading bool) {
	return daemon.service.Snapshot()
}

func (daemon *Daemon) SessionCount() int {
	return daemon.service.SessionCount()
}

func (daemon *Daemon) SessionHub(sessionID string) *SessionUpdateHub {
	return daemon.service.SessionHub(sessionID)
}

func (daemon *Daemon) Reset() {
	if daemon.stopped.Load() {
		return
	}

	daemon.service.Reset()
}

func (daemon *Daemon) Stop() {
	daemon.stop(true)
}

// StopSilently stops the daemon without triggering the stop callback.
// This is used by server shutdown code paths to avoid recursive stop calls.
func (daemon *Daemon) StopSilently() {
	daemon.stop(false)
}

// SetOnStop sets a callback invoked when the daemon stops itself.
func (daemon *Daemon) SetOnStop(callback func()) {
	daemon.mu.Lock()
	daemon.onStop = callback
	daemon.mu.Unlock()
}

func (daemon *Daemon) stop(notify bool) {
	if !daemon.stopped.CompareAndSwap(false, true) {
		return
	}

	daemon.mu.Lock()
	daemon.cancelIdleStopLocked()
	callback := daemon.onStop
	daemon.mu.Unlock()

	if notify && callback != nil {
		callback()
	}
}

func (daemon *Daemon) cancelIdleStopLocked() {
	daemon.idleToken++
}

func (daemon *Daemon) scheduleIdleStopLocked() {
	if daemon.idleTimeout <= 0 {
		return
	}

	daemon.idleToken++
	token := daemon.idleToken
	timeout := daemon.idleTimeout

	time.AfterFunc(timeout, func() {
		// Token check makes timer cancellation lock-free for callers.
		daemon.mu.Lock()
		if daemon.stopped.Load() || daemon.idleToken != token {
			daemon.mu.Unlock()
			return
		}
		daemon.mu.Unlock()

		if daemon.sessions.Count() == 0 {
			daemon.Stop()
		}
	})
}

func (daemon *Daemon) registerSessionPID(request RenderRequest) {
	pid, ok := parseSessionPID(request.SessionID)
	if !ok {
		return
	}

	var shellName string
	if request.Flags != nil {
		shellName = request.Flags.Shell
	}

	daemon.sessions.Register(pid, "", shellName)

	// Active tracked PID means daemon must not stop for idleness.
	daemon.mu.Lock()
	daemon.cancelIdleStopLocked()
	daemon.mu.Unlock()
}

func (daemon *Daemon) onSessionUnregister(pid int) {
	daemon.service.CompleteSession(strconv.Itoa(pid))
}

func (daemon *Daemon) onAllSessionsEnded() {
	// Called from SessionManager while its lock is held; avoid re-entering sessions locks here.
	daemon.mu.Lock()
	daemon.scheduleIdleStopLocked()
	daemon.mu.Unlock()
}

func (daemon *Daemon) scheduleIdleIfNoSessions() {
	if daemon.sessions.Count() != 0 {
		return
	}

	daemon.mu.Lock()
	daemon.scheduleIdleStopLocked()
	daemon.mu.Unlock()
}

func parseSessionPID(sessionID string) (int, bool) {
	pid, err := strconv.Atoi(sessionID)
	if err != nil {
		return 0, false
	}

	if pid <= 0 {
		return 0, false
	}

	return pid, true
}
