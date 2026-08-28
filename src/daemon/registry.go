package daemon

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/po1o/prompto/src/log"
	"github.com/po1o/prompto/src/prompt"
	"github.com/po1o/prompto/src/runtime"
)

type engineFactory func(flags *runtime.Flags) *prompt.Engine

type sessionState struct {
	engine *prompt.Engine
	// activeCtx/activeCancel/activeID describe the currently active render generation.
	activeCtx    context.Context
	activeCancel context.CancelFunc
	activeID     uint64
}

// EngineRegistry stores prompt engines per session and tracks active renders.
// It supports stream reattach by returning the same engine for a session.
type EngineRegistry struct {
	factory  engineFactory
	sessions map[string]*sessionState
	mu       sync.Mutex
	nextID   atomic.Uint64
}

func NewEngineRegistry(factory engineFactory) *EngineRegistry {
	if factory == nil {
		factory = prompt.New
	}

	return &EngineRegistry{
		factory:  factory,
		sessions: make(map[string]*sessionState),
	}
}

func (registry *EngineRegistry) GetOrCreateEngine(sessionID string, flags *runtime.Flags) *prompt.Engine {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	state, ok := registry.sessions[sessionID]
	if ok && engineServesConfig(state.engine, flags) {
		return state.engine
	}

	if ok {
		// The session is keyed by the client's pid, and the OS reuses pids, so
		// a new shell can land on a dead one's entry. Nothing else reconciles
		// the two: applyRenderFlags refreshes flags and env but never the
		// config, so without this the first config a session rendered would
		// keep rendering for the life of the daemon — a console config
		// answering a desktop session, or the reverse.
		log.Debugf("session %s changed config to %s, rebuilding engine", sessionID, flags.ConfigPath)
	}

	engine := registry.factory(flags)
	registry.sessions[sessionID] = &sessionState{
		engine: engine,
	}

	return engine
}

// engineServesConfig reports whether the cached engine was built for the config
// this request names. An empty request path means "whatever the daemon was
// started with", which the server has already filled in, so it is only ever
// empty here in tests.
func engineServesConfig(engine *prompt.Engine, flags *runtime.Flags) bool {
	if engine == nil || flags == nil || flags.ConfigPath == "" {
		return true
	}

	if engine.LayoutConfig == nil {
		return true
	}

	return engine.LayoutConfig.Source == flags.ConfigPath
}

// RenderHandle is one render generation for a session: its engine, the
// context that is cancelled when the generation is superseded, and the
// render ID used to match updates and completions.
type RenderHandle struct {
	// Context is cancelled when this render generation is superseded.
	Context    context.Context
	Engine     *prompt.Engine
	registry   *EngineRegistry
	sessionID  string
	renderID   uint64
	Reattached bool
}

func (h *RenderHandle) Complete() {
	if h == nil || h.registry == nil {
		return
	}

	h.registry.CancelRenderIf(h.sessionID, h.renderID)
}

func (h *RenderHandle) RenderID() uint64 {
	if h == nil {
		return 0
	}

	return h.renderID
}

// StartRender begins, or for a soft cancel reattaches to, the active render
// for a session. A soft cancel (vim toggle) reuses the in-flight render and
// its context so the running computation is preserved; a hard cancel (new
// command) aborts the prior render and starts a fresh generation. See
// cancel.go and ARCHITECTURE.md ("The cancel model").
func (registry *EngineRegistry) StartRender(sessionID string, flags *runtime.Flags, kind CancelKind) *RenderHandle {
	engine := registry.GetOrCreateEngine(sessionID, flags)

	if kind.Repaint() {
		if ctx, renderID, ok := registry.GetActiveRender(sessionID); ok {
			return &RenderHandle{
				Context:    ctx,
				Engine:     engine,
				registry:   registry,
				sessionID:  sessionID,
				renderID:   renderID,
				Reattached: true,
			}
		}
	} else {
		// A hard cancel aborts the prior generation before starting a new one.
		registry.CancelActiveRender(sessionID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	renderID, _ := registry.SetActiveRender(sessionID, ctx, cancel)
	return &RenderHandle{
		Context:    ctx,
		Engine:     engine,
		registry:   registry,
		sessionID:  sessionID,
		renderID:   renderID,
		Reattached: false,
	}
}

func (registry *EngineRegistry) SetActiveRenderCancel(sessionID string, cancel context.CancelFunc) {
	_, _ = registry.SetActiveRender(sessionID, context.Background(), cancel)
}

func (registry *EngineRegistry) SetActiveRender(sessionID string, ctx context.Context, cancel context.CancelFunc) (uint64, bool) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	state, ok := registry.sessions[sessionID]
	if !ok {
		return 0, false
	}

	id := registry.nextID.Add(1)
	state.activeCtx = ctx
	state.activeCancel = cancel
	state.activeID = id
	return id, true
}

func (registry *EngineRegistry) GetActiveRenderContext(sessionID string) (context.Context, bool) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	state, ok := registry.sessions[sessionID]
	if !ok || state.activeCtx == nil {
		return nil, false
	}

	return state.activeCtx, true
}

func (registry *EngineRegistry) GetActiveRender(sessionID string) (context.Context, uint64, bool) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	state, ok := registry.sessions[sessionID]
	if !ok || state.activeCtx == nil {
		return nil, 0, false
	}

	return state.activeCtx, state.activeID, true
}

// CancelActiveRender cancels the active render for a session.
// Repaint requests should skip this cancellation and reattach.
func (registry *EngineRegistry) CancelActiveRender(sessionID string) {
	registry.mu.Lock()

	state, ok := registry.sessions[sessionID]
	if !ok || state.activeCancel == nil {
		registry.mu.Unlock()
		return
	}

	cancel := registry.clearActiveLocked(state)
	registry.mu.Unlock()
	cancel()
}

func (registry *EngineRegistry) CancelRenderIf(sessionID string, renderID uint64) {
	registry.mu.Lock()

	state, ok := registry.sessions[sessionID]
	if !ok || state.activeCancel == nil || state.activeID != renderID {
		registry.mu.Unlock()
		return
	}

	cancel := registry.clearActiveLocked(state)
	registry.mu.Unlock()
	cancel()
}

func (registry *EngineRegistry) ClearActiveRenderIf(sessionID string, renderID uint64) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	state, ok := registry.sessions[sessionID]
	if !ok {
		return
	}

	if state.activeCancel == nil {
		return
	}

	if state.activeID != renderID {
		return
	}

	_ = registry.clearActiveLocked(state)
}

func (registry *EngineRegistry) RemoveSession(sessionID string) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	delete(registry.sessions, sessionID)
}

func (registry *EngineRegistry) Reset() {
	registry.mu.Lock()
	registry.sessions = make(map[string]*sessionState)
	registry.mu.Unlock()
}

func (registry *EngineRegistry) clearActiveLocked(state *sessionState) context.CancelFunc {
	if state == nil {
		return nil
	}

	cancel := state.activeCancel
	state.activeCtx = nil
	state.activeCancel = nil
	state.activeID = 0
	return cancel
}
