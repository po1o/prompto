package daemon

import (
	"context"
	"sync"

	runtimePkg "github.com/po1o/prompto/src/runtime"

	"github.com/po1o/prompto/src/prompt"
	"github.com/po1o/prompto/src/template"
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

type bundleOptions struct {
	includeSecondary bool
	includeTransient bool
}

type promptBundleRenderer interface {
	Bundle(*prompt.Engine, string, bundleOptions) PromptBundle
}

type defaultPromptBundleRenderer struct{}

func (renderer defaultPromptBundleRenderer) Bundle(engine *prompt.Engine, primary string, options bundleOptions) PromptBundle {
	if engine == nil {
		return PromptBundle{}
	}

	bundle := PromptBundle{
		Primary: primary,
		RPrompt: engine.StreamingRPrompt(),
	}

	if options.includeTransient {
		bundle.Transient = engine.StreamingTransientPrompt()
		bundle.RTransient = engine.StreamingTransientRPrompt()
	}

	if options.includeSecondary {
		bundle.Secondary = engine.ExtraPromptNoReset(prompt.Secondary)
	}

	return bundle
}

// RenderPipeline owns per-session render execution end to end: reload gating,
// engine reuse and cancellation (via EngineRegistry), per-session update hubs,
// and turning engine state into client-facing bundles. It absorbs what were
// previously the SessionRenderRuntime and RequestManager layers.
type RenderPipeline struct {
	// gate blocks new requests during reload and waits for active requests.
	gate *ReloadGate
	// registry reuses/cancels per-session engines and active renders.
	registry *EngineRegistry
	// sessions stores the update hub for each session ID.
	sessions *PromptSessionStore
	// renderer turns engine state into bundle text sent to clients.
	renderer promptBundleRenderer
	// deviceCache is injected into each engine before rendering.
	deviceCache prompt.DeviceCache
}

func NewRenderPipeline(registry *EngineRegistry, gate *ReloadGate, renderer promptBundleRenderer, deviceCache prompt.DeviceCache) *RenderPipeline {
	if gate == nil {
		gate = NewReloadGate()
	}

	if renderer == nil {
		renderer = defaultPromptBundleRenderer{}
	}

	return &RenderPipeline{
		gate:        gate,
		registry:    registry,
		sessions:    NewPromptSessionStore(registry),
		renderer:    renderer,
		deviceCache: deviceCache,
	}
}

// ActiveRender is one in-flight render stream for a session. It unifies the
// former RenderHandle / RequestHandle / SessionRenderHandle plumbing: the
// render generation (engine, context, render ID, reattach flag), the update
// hub and its relay, and the reload-gate release. Complete is idempotent.
type ActiveRender struct {
	render        *RenderHandle
	relay         *StreamRelay
	hub           *SessionUpdateHub
	renderer      promptBundleRenderer
	releaseActive func()
	once          sync.Once
}

func (active *ActiveRender) engine() *prompt.Engine {
	if active == nil || active.render == nil {
		return nil
	}

	return active.render.Engine
}

func (active *ActiveRender) context() context.Context {
	if active == nil || active.render == nil {
		return nil
	}

	return active.render.Context
}

func (active *ActiveRender) renderID() uint64 {
	if active == nil || active.render == nil {
		return 0
	}

	return active.render.RenderID()
}

func (active *ActiveRender) reattached() bool {
	if active == nil || active.render == nil {
		return false
	}

	return active.render.Reattached
}

func (pipeline *RenderPipeline) newActiveRender(sessionID string, flags *runtimePkg.Flags, kind CancelKind) *ActiveRender {
	release := pipeline.gate.StartRequest()
	render := pipeline.registry.StartRender(sessionID, flags, kind)
	hub := pipeline.sessions.Hub(sessionID)

	return &ActiveRender{
		render:        render,
		relay:         NewStreamRelay(hub),
		hub:           hub,
		renderer:      pipeline.renderer,
		releaseActive: release,
	}
}

func (pipeline *RenderPipeline) Start(sessionID string, flags *runtimePkg.Flags, kind CancelKind) (PromptBundle, *ActiveRender) {
	repaint := kind.Repaint()
	active := pipeline.newActiveRender(sessionID, flags, kind)
	engine := active.engine()
	primary := ""

	if repaint && !active.reattached() && (engine == nil || engine.Config == nil) {
		active.Complete()
		bundle := pipeline.renderer.Bundle(engine, primary, bundleOptions{})
		return bundle, nil
	}

	if engine != nil && engine.Config != nil {
		engine.SetDeviceCache(pipeline.deviceCache)
		applyRenderFlags(engine, flags, repaint)
		template.Init(engine.Env, engine.Config.Var, engine.Config.Maps)

		if flags != nil && flags.Type != "" && flags.Type != prompt.PRIMARY {
			// Non-primary type requests are synchronous one-shots.
			bundle := renderPromptByType(engine, flags.Type, flags.Command)
			if active.hub != nil {
				active.hub.Publish(renderCompletePayload, active.renderID())
			}
			return bundle, active
		}

		if repaint && active.reattached() {
			// Repaint updates vim-mode-driven output without restarting async segment jobs.
			primary = engine.PrimaryRepaint()
			if engine.PendingSegmentCount() == 0 && active.hub != nil {
				active.hub.Publish(renderCompletePayload, active.renderID())
			}

			bundle := pipeline.renderer.Bundle(engine, primary, bundleOptions{includeTransient: true})
			return bundle, active
		}

		if repaint {
			primary = engine.PrimaryRepaint()
			active.Complete()

			bundle := pipeline.renderer.Bundle(engine, primary, bundleOptions{includeTransient: true})
			return bundle, nil
		}

		timeout := engine.Config.GetDaemonTimeout()
		if active.hub != nil {
			renderID := active.renderID()
			// PrimaryStreaming returns quickly with pending placeholders, then publishes updates.
			primary = engine.PrimaryStreaming(active.context(), timeout, func(segmentName string) {
				if segmentName == "" {
					active.hub.Publish(renderCompletePayload, renderID)
					return
				}

				active.hub.Publish(segmentName, renderID)
			})

			if engine.PendingSegmentCount() == 0 {
				active.hub.Publish(renderCompletePayload, renderID)
				active.Complete()

				bundle := pipeline.renderer.Bundle(engine, primary, bundleOptions{
					includeSecondary: true,
					includeTransient: true,
				})
				return bundle, nil
			}
		} else {
			primary = engine.Primary()

			bundle := pipeline.renderer.Bundle(engine, primary, bundleOptions{
				includeSecondary: true,
				includeTransient: true,
			})
			return bundle, nil
		}
	}

	options := bundleOptions{}
	if flags == nil || flags.Type == "" || flags.Type == prompt.PRIMARY {
		options.includeTransient = true
	}

	bundle := pipeline.renderer.Bundle(engine, primary, options)
	return bundle, active
}

// Reload runs action while holding the reload gate, after waiting for active
// requests to drain and blocking new ones until it returns.
func (pipeline *RenderPipeline) Reload(action func()) {
	pipeline.gate.BeginReload()
	defer pipeline.gate.EndReload()

	if action == nil {
		return
	}

	action()
}

func (pipeline *RenderPipeline) Snapshot() (active int, reloading bool) {
	return pipeline.gate.Snapshot()
}

func (pipeline *RenderPipeline) SessionHub(sessionID string) *SessionUpdateHub {
	return pipeline.sessions.Hub(sessionID)
}

func (pipeline *RenderPipeline) RemoveSession(sessionID string) {
	pipeline.sessions.RemoveSession(sessionID)
}

func (pipeline *RenderPipeline) Reset() {
	if pipeline.sessions == nil {
		return
	}

	pipeline.sessions.Reset()
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
	currentFlags.IsPrimary = currentFlags.Type == "" || currentFlags.Type == prompt.PRIMARY
	if engine.Config != nil {
		currentFlags.HasExtra = engine.Config.HasSecondary || engine.Config.HasTransient ||
			engine.Config.ValidLine != nil || engine.Config.ErrorLine != nil || engine.Config.DebugPrompt != nil
	}

	term, ok := engine.Env.(*runtimePkg.Terminal)
	if !ok {
		return
	}

	term.Init(currentFlags)
}

func (active *ActiveRender) Next(updateContext context.Context, after uint64) (PromptUpdate, bool) {
	if active == nil || active.relay == nil || active.renderer == nil {
		return PromptUpdate{}, false
	}

	if updateContext == nil {
		updateContext = context.Background()
	}

	relayContext := updateContext
	renderContext := active.context()
	if renderContext != nil {
		var cancel context.CancelFunc
		relayContext, cancel = context.WithCancel(updateContext)
		stopCancel := context.AfterFunc(renderContext, cancel)
		defer stopCancel()
		defer cancel()
	}

	snapshot, ok := active.relay.Next(relayContext, after, active.renderID())
	if !ok {
		return PromptUpdate{}, false
	}

	if err := updateContext.Err(); err != nil {
		return PromptUpdate{}, false
	}

	if renderContext != nil {
		if err := renderContext.Err(); err != nil {
			return PromptUpdate{}, false
		}
	}

	engine := active.engine()
	if engine == nil {
		return PromptUpdate{}, false
	}

	primary := ""
	if engine.Config != nil {
		primary = engine.ReRender()
	}

	return PromptUpdate{
		Snapshot: snapshot,
		Bundle: active.renderer.Bundle(engine, primary, bundleOptions{
			includeSecondary: snapshot.Payload == renderCompletePayload,
			includeTransient: true,
		}),
	}, true
}

func (active *ActiveRender) Complete() {
	if active == nil {
		return
	}

	active.once.Do(func() {
		if engine := active.engine(); engine != nil {
			// Detach callback so old/canceled renders stop publishing updates.
			ClearSegmentUpdates(engine)
		}

		if active.render != nil {
			active.render.Complete()
		}

		if active.releaseActive != nil {
			active.releaseActive()
		}
	})
}
