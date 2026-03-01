package daemon

import (
	"context"

	runtimePkg "github.com/jandedobbeleer/oh-my-posh/src/runtime"

	"github.com/jandedobbeleer/oh-my-posh/src/prompt"
)

type PromptBundle struct {
	Primary   string
	RPrompt   string
	Secondary string
	Transient string
}

type PromptUpdate struct {
	Snapshot UpdateSnapshot
	Bundle   PromptBundle
}

type promptBundleRenderer interface {
	Render(*prompt.Engine, bool) PromptBundle
}

type defaultPromptBundleRenderer struct{}

func (renderer defaultPromptBundleRenderer) Render(engine *prompt.Engine, _ bool) PromptBundle {
	if engine == nil {
		return PromptBundle{}
	}

	return PromptBundle{
		Primary:   engine.Primary(),
		RPrompt:   engine.RPrompt(),
		Secondary: engine.ExtraPrompt(prompt.Secondary),
		Transient: engine.ExtraPrompt(prompt.Transient),
	}
}

type RenderPipeline struct {
	runtime  *SessionRenderRuntime
	renderer promptBundleRenderer
}

func NewRenderPipeline(sessionRuntime *SessionRenderRuntime, renderer promptBundleRenderer) *RenderPipeline {
	if renderer == nil {
		renderer = defaultPromptBundleRenderer{}
	}

	return &RenderPipeline{
		runtime:  sessionRuntime,
		renderer: renderer,
	}
}

type ActiveRender struct {
	handle   *SessionRenderHandle
	renderer promptBundleRenderer
}

func (pipeline *RenderPipeline) Start(sessionID string, flags *runtimePkg.Flags, repaint bool) (PromptBundle, *ActiveRender) {
	handle := pipeline.runtime.StartRequest(sessionID, flags, repaint)
	bundle := pipeline.renderer.Render(handle.Engine(), repaint)
	return bundle, &ActiveRender{
		handle:   handle,
		renderer: pipeline.renderer,
	}
}

func (active *ActiveRender) Next(updateContext context.Context, after uint64) (PromptUpdate, bool) {
	if active == nil || active.handle == nil || active.handle.Relay() == nil || active.renderer == nil {
		return PromptUpdate{}, false
	}

	snapshot, ok := active.handle.Relay().Next(updateContext, after)
	if !ok {
		return PromptUpdate{}, false
	}

	return PromptUpdate{
		Snapshot: snapshot,
		Bundle:   active.renderer.Render(active.handle.Engine(), true),
	}, true
}

func (active *ActiveRender) Complete() {
	if active == nil || active.handle == nil {
		return
	}

	active.handle.Complete()
}
