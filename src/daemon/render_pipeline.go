package daemon

import (
	"context"

	runtimePkg "github.com/po1o/prompto/src/runtime"

	"github.com/po1o/prompto/src/prompt"
)

type PromptBundle struct {
	Extras     map[string]string
	Primary    string
	RPrompt    string
	RTransient string
	Secondary  string
	Transient  string
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
		Primary:    primary,
		RPrompt:    engine.StreamingRPrompt(),
		RTransient: engine.TransientRPrompt(),
		Secondary:  engine.ExtraPrompt(prompt.Secondary),
		Transient:  engine.ExtraPrompt(prompt.Transient),
	}
}

type RenderPipeline struct {
	// runtime gives access to session-scoped engine/hub/request state.
	runtime *SessionRenderRuntime
	// renderer turns engine state into bundle text sent to clients.
	renderer promptBundleRenderer
	// deviceCache is injected into each engine before rendering.
	deviceCache prompt.DeviceCache
}

func NewRenderPipeline(sessionRuntime *SessionRenderRuntime, renderer promptBundleRenderer, deviceCache prompt.DeviceCache) *RenderPipeline {
	if renderer == nil {
		renderer = defaultPromptBundleRenderer{}
	}

	return &RenderPipeline{
		runtime:     sessionRuntime,
		renderer:    renderer,
		deviceCache: deviceCache,
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
		engine.SetDeviceCache(pipeline.deviceCache)
		applyRenderFlags(engine, flags, repaint)

		if flags != nil && flags.Type != "" && flags.Type != prompt.PRIMARY {
			// Non-primary type requests are synchronous one-shots.
			bundle := renderPromptByType(engine, flags.Type, flags.Command)
			if handle.Hub() != nil {
				handle.Hub().Publish(renderCompletePayload, handle.RenderID())
			}
			return bundle, &ActiveRender{
				handle:   handle,
				renderer: pipeline.renderer,
			}
		}

		if repaint && handle.Reattached() {
			// Repaint updates vim-mode-driven output without restarting async segment jobs.
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
			// PrimaryStreaming returns quickly with pending placeholders, then publishes updates.
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

func renderPromptByType(engine *prompt.Engine, promptType, command string) PromptBundle {
	if engine == nil {
		return PromptBundle{}
	}

	text := ""
	switch promptType {
	case prompt.DEBUG:
		text = engine.ExtraPrompt(prompt.Debug)
	case prompt.PRIMARY:
		text = engine.Primary()
	case prompt.SECONDARY:
		text = engine.ExtraPrompt(prompt.Secondary)
	case prompt.TRANSIENT:
		text = engine.ExtraPrompt(prompt.Transient)
	case prompt.RIGHT:
		text = engine.RPrompt()
	case prompt.TOOLTIP:
		text = engine.Tooltip(command)
	case prompt.VALID:
		text = engine.ExtraPrompt(prompt.Valid)
	case prompt.ERROR:
		text = engine.ExtraPrompt(prompt.Error)
	case prompt.PREVIEW:
		text = engine.Preview()
	default:
		return PromptBundle{}
	}

	return PromptBundle{
		Extras: map[string]string{
			promptType: text,
		},
	}
}

func applyRenderFlags(engine *prompt.Engine, flags *runtimePkg.Flags, repaint bool) {
	if engine == nil || flags == nil || engine.Env == nil {
		return
	}

	currentFlags := engine.Env.Flags()
	if currentFlags == nil {
		return
	}

	if repaint {
		// Repaint only needs VimMode change; keep previous request context/flags intact.
		currentFlags.VimMode = flags.VimMode
		return
	}

	*currentFlags = *flags

	term, ok := engine.Env.(*runtimePkg.Terminal)
	if !ok {
		return
	}

	term.Init(currentFlags)
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
