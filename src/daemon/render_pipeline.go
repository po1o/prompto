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
	Bundle   PromptBundle
	Snapshot UpdateSnapshot
}

type promptBundleRenderer interface {
	Bundle(*prompt.Engine, string) PromptBundle
}

type defaultPromptBundleRenderer struct{}

func (renderer defaultPromptBundleRenderer) Bundle(engine *prompt.Engine, primary string) PromptBundle {
	if engine == nil {
		return PromptBundle{}
	}

	return PromptBundle{
		Primary:   primary,
		RPrompt:   engine.StreamingRPrompt(),
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
	engine := handle.Engine()
	primary := ""
	if engine != nil && engine.Config != nil {
		if repaint && handle.Reattached() {
			applyRepaintFlags(engine, flags)
			primary = engine.PrimaryRepaint()
			if len(engine.PendingSegments()) == 0 && handle.Hub() != nil {
				handle.Hub().Publish(renderCompletePayload, handle.RenderID())
			}

			bundle := pipeline.renderer.Bundle(engine, primary)
			return bundle, &ActiveRender{
				handle:   handle,
				renderer: pipeline.renderer,
			}
		}

		timeout := engine.Config.GetDaemonTimeout()
		if handle.Hub() != nil {
			renderID := handle.RenderID()
			primary = engine.PrimaryStreaming(handle.Context(), timeout, func(segmentName string) {
				if segmentName == "" {
					handle.Hub().Publish(renderCompletePayload, renderID)
					return
				}

				handle.Hub().Publish(segmentName, renderID)
			})

			if len(engine.PendingSegments()) == 0 {
				handle.Hub().Publish(renderCompletePayload, renderID)
			}
		} else {
			primary = engine.Primary()
		}
	}

	bundle := pipeline.renderer.Bundle(engine, primary)
	return bundle, &ActiveRender{
		handle:   handle,
		renderer: pipeline.renderer,
	}
}

func applyRepaintFlags(engine *prompt.Engine, flags *runtimePkg.Flags) {
	if engine == nil || flags == nil || engine.Env == nil {
		return
	}

	currentFlags := engine.Env.Flags()
	if currentFlags == nil {
		return
	}

	currentFlags.VimMode = flags.VimMode
}

func (active *ActiveRender) Next(updateContext context.Context, after uint64) (PromptUpdate, bool) {
	if active == nil || active.handle == nil || active.handle.Relay() == nil || active.renderer == nil {
		return PromptUpdate{}, false
	}

	snapshot, ok := active.handle.Relay().Next(updateContext, after, active.handle.RenderID())
	if !ok {
		return PromptUpdate{}, false
	}

	engine := active.handle.Engine()
	if engine == nil {
		return PromptUpdate{}, false
	}

	primary := ""
	if engine.Config != nil {
		primary = engine.ReRender()
	}

	return PromptUpdate{
		Snapshot: snapshot,
		Bundle:   active.renderer.Bundle(engine, primary),
	}, true
}

func (active *ActiveRender) Complete() {
	if active == nil || active.handle == nil {
		return
	}

	active.handle.Complete()
}
