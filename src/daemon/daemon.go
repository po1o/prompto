package daemon

import (
	"context"
	"maps"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/prompt"
	"github.com/po1o/prompto/src/runtime"
)

// renderCompletePayload is published on a session's update hub to signal that
// the active render generation has produced its final state. Server.RenderPrompt
// and Daemon.NextUpdate use it to terminate the response stream.
const renderCompletePayload = "__prompto_render_complete__"

// RenderRequest is the daemon's render entry point — what the server sends to
// Daemon.StartRender per RPC.
type RenderRequest struct {
	Flags     *runtime.Flags
	SessionID string
	// Cancel selects the cancel semantics for this render: CancelHard
	// (default — new command, abort prior in-flight work) or CancelSoft
	// (vim toggle / repaint — preserve in-flight work and reattach).
	Cancel CancelKind
}

// RenderResponse is the daemon's render exit point — initial bundles and
// per-segment updates returned to the server, which forwards them over gRPC.
type RenderResponse struct {
	Bundle   PromptBundle
	Type     string
	Segment  string
	Sequence uint64
}

// Daemon owns the long-lived prompt-rendering process: lifecycle (lock file,
// idle shutdown, on-stop callback), per-shell PID tracking, the device cache,
// and the per-session active-render map. Render execution itself is delegated
// to RenderPipeline. See ARCHITECTURE.md.
type Daemon struct {
	done                  chan struct{}
	onStop                func()
	deviceCache           *DeviceCache
	renders               map[string]*ActiveRender
	segmentToggles        map[string]map[string]bool
	configWatcher         *ConfigWatcher
	binaryWatcher         *BinaryWatcher
	configReloadCh        chan struct{}
	sessions              *ProcessTracker
	pipeline              *RenderPipeline
	configPath            string
	idleTimeout           time.Duration
	idleToken             uint64
	lastConfigModUnixNano atomic.Int64
	toggleMu              sync.RWMutex
	mu                    sync.Mutex
	rendersMu             sync.Mutex
	reloadMu              sync.Mutex
	stopped               atomic.Bool
}

func New(renderer promptBundleRenderer) *Daemon {
	return NewWithIdleTimeoutAndDeviceCache(5*time.Minute, renderer, nil)
}

func NewFromConfig(configPath string, renderer promptBundleRenderer) *Daemon {
	return NewFromConfigWithDeviceCache(configPath, renderer, nil)
}

func NewWithIdleTimeout(idleTimeout time.Duration, renderer promptBundleRenderer) *Daemon {
	return NewWithIdleTimeoutAndDeviceCache(idleTimeout, renderer, nil)
}

func NewFromConfigWithDeviceCache(configPath string, renderer promptBundleRenderer, deviceCache *DeviceCache) *Daemon {
	cfg := config.Load(configPath)
	daemon := NewWithIdleTimeoutAndDeviceCache(cfg.GetDaemonIdleTimeout(), renderer, deviceCache)
	daemon.configPath = configPath
	daemon.startReloadAndWatchers()
	return daemon
}

func NewWithIdleTimeoutAndDeviceCache(idleTimeout time.Duration, renderer promptBundleRenderer, deviceCache *DeviceCache) *Daemon {
	if deviceCache == nil {
		deviceCache = NewDeviceCache()
	}

	registry := NewEngineRegistry(prompt.New)
	gate := NewReloadGate()
	pipeline := NewRenderPipeline(registry, gate, renderer, newPromptDeviceCacheBridge(deviceCache))

	daemon := &Daemon{
		pipeline:       pipeline,
		deviceCache:    deviceCache,
		renders:        make(map[string]*ActiveRender),
		segmentToggles: make(map[string]map[string]bool),
		configReloadCh: make(chan struct{}, 1),
		done:           make(chan struct{}),
		idleTimeout:    idleTimeout,
	}
	daemon.sessions = NewProcessTracker(daemon.onSessionUnregister, daemon.onAllSessionsEnded)

	// Start the idle timer immediately; it is canceled on first tracked render.
	daemon.mu.Lock()
	daemon.scheduleIdleStopLocked()
	daemon.mu.Unlock()

	return daemon
}

func (daemon *Daemon) DeviceCache() *DeviceCache {
	return daemon.deviceCache
}

// ConfigPath returns the resolved root config path the daemon is watching, or
// the empty string if the daemon was constructed without a config path.
func (daemon *Daemon) ConfigPath() string {
	return daemon.configPath
}

func (daemon *Daemon) StartRender(request RenderRequest) RenderResponse {
	if daemon.stopped.Load() {
		return RenderResponse{Type: "stopped"}
	}

	// Any tracked PID render is considered activity and cancels pending idle stop.
	daemon.registerSessionPID(request)

	daemon.rendersMu.Lock()
	existing, ok := daemon.renders[request.SessionID]
	var staleCompleted *ActiveRender
	var previous *ActiveRender
	if ok && existing != nil && daemon.isCompletedRender(existing) {
		delete(daemon.renders, request.SessionID)
		staleCompleted = existing
		existing = nil
		ok = false
	}

	if ok && existing != nil && !request.Cancel.Repaint() {
		// A hard cancel starts a new render generation; cancel the previous one.
		delete(daemon.renders, request.SessionID)
		previous = existing
	}
	daemon.rendersMu.Unlock()

	if staleCompleted != nil {
		staleCompleted.Complete()
	}
	if previous != nil {
		previous.Complete()
	}

	bundle, active := daemon.pipeline.Start(request.SessionID, request.Flags, request.Cancel)
	sequence := daemon.currentSequence(request.SessionID)

	daemon.rendersMu.Lock()
	if active == nil {
		delete(daemon.renders, request.SessionID)
	} else {
		daemon.renders[request.SessionID] = active
	}
	daemon.rendersMu.Unlock()

	return RenderResponse{
		Type:     "initial",
		Bundle:   bundle,
		Sequence: sequence,
	}
}

func (daemon *Daemon) NextUpdate(ctx context.Context, sessionID string, after uint64) (RenderResponse, bool) {
	if daemon.stopped.Load() {
		return RenderResponse{}, false
	}

	daemon.rendersMu.Lock()
	active, ok := daemon.renders[sessionID]
	daemon.rendersMu.Unlock()
	if !ok || active == nil {
		return RenderResponse{}, false
	}

	update, ok := active.Next(ctx, after)
	if !ok {
		if ctx != nil && ctx.Err() != nil {
			return RenderResponse{}, false
		}

		daemon.releaseActiveRenderIfCurrent(sessionID, active)
		return RenderResponse{}, false
	}

	if update.Snapshot.Payload == renderCompletePayload {
		daemon.releaseActiveRenderIfCurrent(sessionID, active)
	}

	return RenderResponse{
		Type:     "update",
		Sequence: update.Snapshot.Sequence,
		Segment:  update.Snapshot.Payload,
		Bundle:   update.Bundle,
	}, true
}

func (daemon *Daemon) CompleteSession(sessionID string) {
	if daemon.stopped.Load() {
		return
	}

	daemon.completeRender(sessionID)

	pid, ok := parseSessionPID(sessionID)
	if ok {
		// PID-backed sessions are lifecycle-managed by ProcessTracker callbacks.
		daemon.sessions.Unregister(pid)
		return
	}

	daemon.scheduleIdleIfNoSessions()
}

func (daemon *Daemon) Reload(action func()) {
	if daemon.stopped.Load() {
		return
	}

	daemon.pipeline.Reload(action)
}

func (daemon *Daemon) Snapshot() (active int, reloading bool) {
	return daemon.pipeline.Snapshot()
}

func (daemon *Daemon) SessionCount() int {
	daemon.rendersMu.Lock()
	defer daemon.rendersMu.Unlock()

	return len(daemon.renders)
}

func (daemon *Daemon) SessionHub(sessionID string) *SessionUpdateHub {
	return daemon.pipeline.SessionHub(sessionID)
}

func (daemon *Daemon) Reset() {
	if daemon.stopped.Load() {
		return
	}

	daemon.rendersMu.Lock()
	activeRenders := make([]*ActiveRender, 0, len(daemon.renders))
	for sessionID, active := range daemon.renders {
		activeRenders = append(activeRenders, active)
		delete(daemon.renders, sessionID)
	}
	daemon.rendersMu.Unlock()

	for _, active := range activeRenders {
		if active == nil {
			continue
		}

		active.Complete()
	}

	if daemon.pipeline == nil {
		return
	}

	daemon.pipeline.Reset()
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

	// Signal the reload-worker goroutine to exit and tear down watchers.
	if daemon.done != nil {
		close(daemon.done)
	}
	if daemon.configWatcher != nil {
		_ = daemon.configWatcher.Close()
	}
	if daemon.binaryWatcher != nil {
		_ = daemon.binaryWatcher.Close()
	}

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
	// ProcessTracker has already removed the PID from its tracking; we only
	// need to tear down the render state for that session ID.
	daemon.completeRender(strconv.Itoa(pid))
}

func (daemon *Daemon) onAllSessionsEnded() {
	// Called from ProcessTracker while its lock is held; avoid re-entering sessions locks here.
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

// completeRender tears down the active render stream for a session and clears
// its pipeline-level state. Used by both CompleteSession (the public API) and
// onSessionUnregister (when a tracked PID exits). It deliberately does NOT
// touch ProcessTracker or idle scheduling — those are the caller's concern.
func (daemon *Daemon) completeRender(sessionID string) {
	daemon.rendersMu.Lock()
	active, ok := daemon.renders[sessionID]
	if ok {
		delete(daemon.renders, sessionID)
	}
	daemon.rendersMu.Unlock()

	if ok && active != nil {
		// Ensure request gate "active" counter is released.
		active.Complete()
	}

	daemon.pipeline.RemoveSession(sessionID)
}

func (daemon *Daemon) releaseActiveRenderIfCurrent(sessionID string, expected *ActiveRender) {
	if expected == nil {
		return
	}

	daemon.rendersMu.Lock()
	current, ok := daemon.renders[sessionID]
	if !ok || current != expected {
		daemon.rendersMu.Unlock()
		return
	}

	delete(daemon.renders, sessionID)
	daemon.rendersMu.Unlock()

	expected.Complete()
}

func (daemon *Daemon) currentSequence(sessionID string) uint64 {
	if daemon.pipeline == nil {
		return 0
	}

	hub := daemon.pipeline.SessionHub(sessionID)
	if hub == nil {
		return 0
	}

	snapshot, ok := hub.Last()
	if !ok {
		return 0
	}

	return snapshot.Sequence
}

func (daemon *Daemon) isCompletedRender(active *ActiveRender) bool {
	if active == nil || active.hub == nil {
		return false
	}

	snapshot, ok := active.hub.Last()
	if !ok {
		return false
	}

	if snapshot.Payload != renderCompletePayload {
		return false
	}

	renderID := active.renderID()
	if snapshot.RenderID == 0 || snapshot.RenderID == renderID {
		return true
	}

	return false
}

// SessionToggles returns the per-session segment-toggle map, seeding from the
// persistent toggle cache on first access. The returned map is a clone;
// callers may mutate it without affecting daemon state.
func (daemon *Daemon) SessionToggles(sessionID string) map[string]bool {
	daemon.toggleMu.RLock()
	existing, ok := daemon.segmentToggles[sessionID]
	daemon.toggleMu.RUnlock()
	if ok {
		return cloneToggleMap(existing)
	}

	baseToggles, _ := cache.Get[map[string]bool](cache.Session, cache.TOGGLECACHE)
	cloned := cloneToggleMap(baseToggles)

	daemon.toggleMu.Lock()
	daemon.segmentToggles[sessionID] = cloned
	daemon.toggleMu.Unlock()

	return cloneToggleMap(cloned)
}

// ToggleSegment flips the visibility of the named segments for a session:
// segments currently toggled-on become toggled-off and vice versa.
func (daemon *Daemon) ToggleSegment(sessionID string, segments []string) {
	current := daemon.SessionToggles(sessionID)

	for _, segment := range segments {
		if current[segment] {
			delete(current, segment)
			continue
		}

		current[segment] = true
	}

	daemon.toggleMu.Lock()
	daemon.segmentToggles[sessionID] = current
	daemon.toggleMu.Unlock()
}

// ResetToggles clears all per-session segment toggles. Used when the
// persistent cache is cleared.
func (daemon *Daemon) ResetToggles() {
	daemon.toggleMu.Lock()
	daemon.segmentToggles = make(map[string]map[string]bool)
	daemon.toggleMu.Unlock()
}

func cloneToggleMap(source map[string]bool) map[string]bool {
	if len(source) == 0 {
		return map[string]bool{}
	}

	cloned := make(map[string]bool, len(source))
	maps.Copy(cloned, source)

	return cloned
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
