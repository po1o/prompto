package daemon

import (
	"context"
	"sync"

	"github.com/jandedobbeleer/oh-my-posh/src/prompt"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime"
)

type engineFactory func(flags *runtime.Flags) *prompt.Engine

type sessionState struct {
	engine       *prompt.Engine
	activeCancel context.CancelFunc
}

// EngineRegistry stores prompt engines per session and tracks active renders.
// It supports stream reattach by returning the same engine for a session.
type EngineRegistry struct {
	factory  engineFactory
	sessions map[string]*sessionState
	mu       sync.Mutex
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
	registry.mu.Lock()
	defer registry.mu.Unlock()

	state, ok := registry.sessions[sessionID]
	if !ok {
		return
	}

	state.activeCancel = cancel
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
	state.activeCancel = nil
}

func (registry *EngineRegistry) RemoveSession(sessionID string) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	delete(registry.sessions, sessionID)
}
