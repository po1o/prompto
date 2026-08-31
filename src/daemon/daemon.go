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
	Flags *runtime.Flags
	// Env holds the client shell's environment variables. Segments resolve
	// Getenv against these instead of the daemon's own (stale) environment.
	Env map[string]string
	// ClientDeadline is when the caller stops listening, as gRPC propagated it
	// from the client's context. Zero when the caller set none. The render uses
	// it to make sure anything it still has to say is said while someone is
	// there to hear it.
	ClientDeadline time.Time
	SessionID      string
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
	seededToggles         map[string]bool
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
	pipeline := NewRenderPipeline(registry, gate, renderer, deviceCache)

	daemon := &Daemon{
		pipeline:       pipeline,
		deviceCache:    deviceCache,
		renders:        make(map[string]*ActiveRender),
		segmentToggles: make(map[string]map[string]bool),
		seededToggles:  configToggleSnapshot(),
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

	// What becomes of the session's current handle, if it has one: a finished
	// generation and a hard cancel both retire it outright, while a soft cancel
	// keeps the generation alive for the reattaching request and only replaces
	// the handle in front of it.
	daemon.rendersMu.Lock()
	existing := daemon.renders[request.SessionID]
	var retire, superseded *ActiveRender
	switch {
	case existing == nil:
	case daemon.isCompletedRender(existing), !request.Cancel.Repaint():
		delete(daemon.renders, request.SessionID)
		retire = existing
	default:
		superseded = existing
	}
	daemon.rendersMu.Unlock()

	retire.Complete()

	bundle, active := daemon.pipeline.Start(request)

	// Retiring the replaced handle after Start, so the reload gate's active
	// count never dips to zero between the two and lets a queued reload cut in
	// on this render. Release, not Complete: the reattached handle shares the
	// generation, which must survive.
	superseded.Release()

	// The render's own baseline, taken before it could publish. Re-reading the
	// hub here instead would skip whatever the render published while it ran —
	// including its completion, leaving the client waiting on a sequence that
	// never arrives while the prompt keeps showing pending placeholders. A
	// render that produced no stream has no baseline of its own, and nothing
	// will follow it, so the hub's current position stands in.
	sequence := daemon.pipeline.SessionHub(request.SessionID).Sequence()
	if active != nil {
		sequence = active.BaseSequence()
	}

	daemon.rendersMu.Lock()
	if active == nil {
		delete(daemon.renders, request.SessionID)
	} else {
		daemon.renders[request.SessionID] = active
	}
	daemon.rendersMu.Unlock()

	return RenderResponse{
		Type:     ResponseTypeInitial,
		Bundle:   bundle,
		Sequence: sequence,
	}
}

func (daemon *Daemon) NextUpdate(ctx context.Context, sessionID string, after uint64) (RenderResponse, bool) {
	if daemon.stopped.Load() {
		return RenderResponse{}, false
	}

	daemon.rendersMu.Lock()
	active := daemon.renders[sessionID]
	daemon.rendersMu.Unlock()
	if active == nil {
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

// completeRender tears down a finished session: the active render stream, its
// pipeline-level state, and its segment toggles. Used by both CompleteSession
// (the public API) and onSessionUnregister (when a tracked PID exits). It
// deliberately does NOT touch ProcessTracker or idle scheduling — those are
// the caller's concern.
func (daemon *Daemon) completeRender(sessionID string) {
	// Sessions are keyed by the shell's pid and the OS recycles pids, so a new
	// shell can land on a dead one's entry. Its toggle set has to go with it:
	// SessionToggles treats a stored map as authoritative even when empty, so
	// an inherited one would leave a `toggled: true` segment visible in a shell
	// that never toggled it. Dropping it also bounds the map's growth.
	daemon.toggleMu.Lock()
	delete(daemon.segmentToggles, sessionID)
	daemon.toggleMu.Unlock()

	daemon.rendersMu.Lock()
	active := daemon.renders[sessionID]
	delete(daemon.renders, sessionID)
	daemon.rendersMu.Unlock()

	// Releases the reload gate's "active" counter as well as the generation.
	active.Complete()
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

func (daemon *Daemon) isCompletedRender(active *ActiveRender) bool {
	if active == nil {
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
	if toggles, seeded := daemon.seededSessionToggles(sessionID); seeded {
		return toggles
	}

	baseToggles, _ := cache.Get[map[string]bool](cache.Session, cache.TOGGLECACHE)
	cloned := cloneToggleMap(baseToggles)

	daemon.toggleMu.Lock()
	defer daemon.toggleMu.Unlock()

	// Another caller may have seeded this session while we were reading the
	// cache — a first render and a `prompto toggle` for the same shell can
	// overlap. Storing our copy unconditionally would discard whatever it had
	// already recorded, silently losing the toggle. First seed wins.
	if seeded, found := daemon.segmentToggles[sessionID]; found {
		cloned = seeded
	}

	daemon.segmentToggles[sessionID] = cloned

	return cloneToggleMap(cloned)
}

// ToggleSegment flips the visibility of the named segments for a session:
// segments currently toggled-on become toggled-off and vice versa.
func (daemon *Daemon) ToggleSegment(sessionID string, segments []string) {
	// Seed first so the session starts from the config's toggles rather than an
	// empty set, then re-read and mutate under the lock: syncConfigToggles can
	// push a newly configured toggle into the stored map in between, and a
	// clone-mutate-store would drop it.
	daemon.SessionToggles(sessionID)

	daemon.toggleMu.Lock()
	defer daemon.toggleMu.Unlock()

	current, ok := daemon.segmentToggles[sessionID]
	if !ok {
		current = map[string]bool{}
		daemon.segmentToggles[sessionID] = current
	}

	for _, segment := range segments {
		if current[segment] {
			delete(current, segment)
			continue
		}

		current[segment] = true
	}
}

// ResetToggles clears all per-session segment toggles. Used when the
// persistent cache is cleared.
func (daemon *Daemon) ResetToggles() {
	daemon.toggleMu.Lock()
	daemon.segmentToggles = make(map[string]map[string]bool)
	// The caller also empties the shared toggle cache this snapshot describes.
	// Left behind, it would report the config's toggles as already seeded and
	// syncConfigToggles would never push them again, so `toggled: true` would
	// stay lost in every live session until the daemon restarted.
	daemon.seededToggles = make(map[string]bool)
	daemon.toggleMu.Unlock()
}

// seededSessionToggles returns a copy of an already-seeded session's toggles.
//
// The copy is taken while the read lock is still held: ToggleSegment and
// syncConfigToggles mutate the stored map in place, so cloning it after
// releasing the lock races them.
func (daemon *Daemon) seededSessionToggles(sessionID string) (map[string]bool, bool) {
	daemon.toggleMu.RLock()
	defer daemon.toggleMu.RUnlock()

	existing, ok := daemon.segmentToggles[sessionID]
	if !ok {
		return nil, false
	}

	return cloneToggleMap(existing), true
}

// syncConfigToggles pushes the toggles the config has gained since the last
// sync into every live session.
//
// A session owns its toggle set once seeded, which is what lets a user switch
// a `toggled: true` segment back on. Left at that, a segment that only just
// gained `toggled: true` would never reach a shell that is already running —
// it would wait for that shell to exit. Comparing the shared cache against the
// last snapshot isolates exactly what the config added, and only that is
// pushed: a toggle the user cleared is absent from the difference, so it is
// not resurrected.
func (daemon *Daemon) syncConfigToggles() {
	if daemon.configPath == "" {
		return
	}

	current, _ := cache.Get[map[string]bool](cache.Session, cache.TOGGLECACHE)

	daemon.toggleMu.Lock()
	defer daemon.toggleMu.Unlock()

	added := make([]string, 0, len(current))
	for name, on := range current {
		if !on || daemon.seededToggles[name] {
			continue
		}

		added = append(added, name)
		daemon.seededToggles[name] = true
	}

	if len(added) == 0 {
		return
	}

	for _, toggles := range daemon.segmentToggles {
		for _, name := range added {
			toggles[name] = true
		}
	}
}

// configToggleSnapshot records the toggles config.Load has already seeded into
// the shared cache, so the first sync reports no difference and does not push
// them into sessions that were seeded from them anyway.
func configToggleSnapshot() map[string]bool {
	seeded, _ := cache.Get[map[string]bool](cache.Session, cache.TOGGLECACHE)
	return cloneToggleMap(seeded)
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
