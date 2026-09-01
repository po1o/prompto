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
	// baseSequence is the hub sequence as of the moment this render was
	// created, before it could publish anything. Streaming from it is what
	// keeps a client from stepping over its own generation's updates.
	baseSequence uint64
	once         sync.Once
}

// newActiveRender is the only constructor and always supplies a render handle,
// so these read it directly. Next is the one entry point a nil ActiveRender can
// reach, and it returns before touching any of them.
func (active *ActiveRender) engine() *prompt.Engine   { return active.render.Engine }
func (active *ActiveRender) context() context.Context { return active.render.Context }
func (active *ActiveRender) renderID() uint64         { return active.render.RenderID() }
func (active *ActiveRender) reattached() bool         { return active.render.Reattached }

// publishSegment announces one finished segment to the session's subscribers.
// PrimaryStreaming names the end of a generation with the empty string. The
// render ID is fixed for the life of the handle, so it needs no capturing.
func (active *ActiveRender) publishSegment(name string) {
	if name == "" {
		name = renderCompletePayload
	}

	active.hub.Publish(name, active.renderID())
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
		// Read before the render runs: everything this generation publishes
		// from here on is still ahead of the client.
		baseSequence: hub.Sequence(),
	}
}

// BaseSequence returns the sequence a client should stream from to receive
// every update this render publishes.
func (active *ActiveRender) BaseSequence() uint64 {
	if active == nil {
		return 0
	}

	return active.baseSequence
}

// isPrimaryRequest reports whether the request wants the primary prompt, the
// only streamed one. Every other type is a synchronous one-shot.
func isPrimaryRequest(flags *runtimePkg.Flags) bool {
	return flags == nil || flags.Type == "" || flags.Type == prompt.PRIMARY
}

func (pipeline *RenderPipeline) Start(sessionID string, flags *runtimePkg.Flags, envVars map[string]string, kind CancelKind) (PromptBundle, *ActiveRender) {
	repaint := kind.Repaint()
	active := pipeline.newActiveRender(sessionID, flags, kind)
	engine := active.engine()

	if engine == nil || engine.Config == nil {
		// Nothing to render from. A repaint that found no live render to
		// reattach to has nowhere to go at all, so it ends here.
		if repaint && !active.reattached() {
			active.Complete()
			return pipeline.renderer.Bundle(engine, "", bundleOptions{}), nil
		}

		options := bundleOptions{includeTransient: isPrimaryRequest(flags)}
		return pipeline.renderer.Bundle(engine, "", options), active
	}

	engine.SetDeviceCache(pipeline.deviceCache)
	applyRenderFlags(engine, flags, envVars, repaint)
	template.Init(engine.Env, engine.Config.Var, engine.Config.Maps)

	if !isPrimaryRequest(flags) {
		// Non-primary type requests are synchronous one-shots.
		bundle := renderPromptByType(engine, flags.Type, flags.Command)
		active.publishSegment("")
		return bundle, active
	}

	if repaint {
		// Repaint updates vim-mode-driven output without restarting async segment jobs.
		primary := engine.PrimaryRepaint()
		bundle := pipeline.renderer.Bundle(engine, primary, bundleOptions{includeTransient: true})

		if !active.reattached() {
			// No in-flight generation to keep streaming from.
			active.Complete()
			return bundle, nil
		}

		if engine.PendingSegmentCount() == 0 {
			active.publishSegment("")
		}

		return bundle, active
	}

	// PrimaryStreaming returns quickly with pending placeholders, then publishes
	// updates. Only a prompt rendered without placeholders is final: asking the
	// engine again for its pending count would race the publisher, which can
	// drain the last result between the render and the question, and this would
	// then hand back a placeholder prompt as the finished one with no stream
	// left to correct it.
	timeout := engine.Config.GetDaemonTimeout()
	primary, streaming := engine.PrimaryStreaming(active.context(), timeout, active.publishSegment)
	if streaming {
		return pipeline.renderer.Bundle(engine, primary, bundleOptions{includeTransient: true}), active
	}

	active.publishSegment("")
	active.Complete()

	bundle := pipeline.renderer.Bundle(engine, primary, bundleOptions{
		includeSecondary: true,
		includeTransient: true,
	})
	return bundle, nil
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

func applyRenderFlags(engine *prompt.Engine, flags *runtimePkg.Flags, envVars map[string]string, repaint bool) {
	if engine == nil || flags == nil || engine.Env == nil {
		return
	}

	// Wrap the engine's Terminal in Environment so segments resolve Getenv
	// against the client request's env vars, not the daemon's own environment.
	daemonEnv, isDaemonEnv := engine.Env.(*Environment)
	if !isDaemonEnv {
		if term, isTerm := engine.Env.(*runtimePkg.Terminal); isTerm {
			daemonEnv = &Environment{Terminal: term}
			engine.Env = daemonEnv
		}
	}

	currentFlags := engine.Env.Flags()
	if currentFlags == nil {
		return
	}

	if repaint {
		// Repaint (soft cancel) only needs VimMode and refreshed env vars;
		// keep previous request context/flags intact.
		if daemonEnv != nil {
			daemonEnv.UpdateForRepaint(flags, envVars)
			return
		}

		currentFlags.VimMode = flags.VimMode
		return
	}

	*currentFlags = *flags
	currentFlags.IsPrimary = currentFlags.Type == "" || currentFlags.Type == prompt.PRIMARY
	if engine.Config != nil {
		currentFlags.HasExtra = engine.Config.HasSecondary || engine.Config.HasTransient ||
			engine.Config.ValidLine != nil || engine.Config.ErrorLine != nil || engine.Config.DebugPrompt != nil
	}

	if daemonEnv == nil {
		return
	}

	daemonEnv.setEnvVars(envVars)
	daemonEnv.Init(currentFlags)
}

func (active *ActiveRender) Next(updateContext context.Context, after uint64) (PromptUpdate, bool) {
	if active == nil {
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
		if active.render != nil {
			active.render.Complete()
		}

		if active.releaseActive != nil {
			active.releaseActive()
		}
	})
}

// Release retires a handle that a soft cancel superseded, returning its
// reload-gate slot without touching the render generation. Complete would
// cancel that generation, and after a soft cancel the reattached handle shares
// its render ID — so completing the old handle would abort the very
// computation the new one exists to reuse. Sharing Complete's once also makes
// a later Complete on this handle a no-op, for the same reason.
func (active *ActiveRender) Release() {
	if active == nil {
		return
	}

	active.once.Do(func() {
		if active.releaseActive != nil {
			active.releaseActive()
		}
	})
}
