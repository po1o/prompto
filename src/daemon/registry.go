package daemon

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/jandedobbeleer/oh-my-posh/src/prompt"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime"
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
	if ok {
		return state.engine
	}

	engine := registry.factory(flags)
	registry.sessions[sessionID] = &sessionState{
		engine: engine,
	}

	return engine
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
