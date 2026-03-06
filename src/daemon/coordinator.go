package daemon

import (
	"context"

	"github.com/jandedobbeleer/oh-my-posh/src/prompt"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime"
)

type RenderHandle struct {
	// Context is canceled when this render generation is superseded.
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

type RenderCoordinator struct {
	registry *EngineRegistry
}

func NewRenderCoordinator(registry *EngineRegistry) *RenderCoordinator {
	return &RenderCoordinator{
		registry: registry,
	}
}

func (c *RenderCoordinator) StartRender(sessionID string, flags *runtime.Flags, repaint bool) *RenderHandle {
	engine := c.registry.GetOrCreateEngine(sessionID, flags)

	if repaint {
		// Repaint keeps ongoing async work alive and reuses the same render generation.
		ctx, renderID, ok := c.registry.GetActiveRender(sessionID)
		if ok {
			return &RenderHandle{
				Engine:     engine,
				Context:    ctx,
				Reattached: true,
				sessionID:  sessionID,
				renderID:   renderID,
				registry:   c.registry,
			}
		}
	}

	if !repaint {
		// A new render request replaces prior work for that session.
		c.registry.CancelActiveRender(sessionID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	renderID, _ := c.registry.SetActiveRender(sessionID, ctx, cancel)
	return &RenderHandle{
		Engine:     engine,
		Context:    ctx,
		Reattached: false,
		sessionID:  sessionID,
		renderID:   renderID,
		registry:   c.registry,
	}
}
