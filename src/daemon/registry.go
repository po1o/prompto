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
	engine       *prompt.Engine
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

// CancelActiveRender cancels the active render for a session.
// Repaint requests should skip this cancellation and reattach.
func (registry *EngineRegistry) CancelActiveRender(sessionID string) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	state, ok := registry.sessions[sessionID]
	if !ok || state.activeCancel == nil {
		return
	}

	state.activeCancel()
	state.activeCtx = nil
	state.activeCancel = nil
	state.activeID = 0
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

	state.activeCtx = nil
	state.activeCancel = nil
	state.activeID = 0
}

func (registry *EngineRegistry) RemoveSession(sessionID string) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	delete(registry.sessions, sessionID)
}
