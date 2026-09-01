package daemon

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/prompt"
	"github.com/po1o/prompto/src/runtime"
	"github.com/po1o/prompto/src/segments/options"
	"github.com/po1o/prompto/src/shell"

	"github.com/stretchr/testify/require"
)

// pendingProbeDelay is how long pendingProbeWriter blocks. Tests drive it to
// land the segment's completion in the window the fixed code has to survive.
var pendingProbeDelay atomic.Int64

// pendingProbeWriter is a segment whose cost the test controls, so a render can
// be made to finish either side of the daemon timeout.
type pendingProbeWriter struct {
	text string
}

func (w *pendingProbeWriter) Enabled() bool {
	time.Sleep(time.Duration(pendingProbeDelay.Load()))
	return true
}

func (w *pendingProbeWriter) Template() string                               { return "READY" }
func (w *pendingProbeWriter) SetText(text string)                            { w.text = text }
func (w *pendingProbeWriter) SetIndex(_ int)                                 {}
func (w *pendingProbeWriter) Text() string                                   { return w.text }
func (w *pendingProbeWriter) Init(_ options.Provider, _ runtime.Environment) {}
func (w *pendingProbeWriter) CacheKey() (string, bool)                       { return "", false }

const (
	pendingProbeType    = config.SegmentType("pending_probe")
	pendingProbeIcon    = "PENDING:"
	pendingProbeTimeout = 20 * time.Millisecond
)

// pendingProbeConfig writes a layout whose only segment is the probe, with a
// distinguishable pending icon so a placeholder prompt is recognisable.
func pendingProbeConfig(t *testing.T) string {
	t.Helper()

	previous, hadPrevious := config.Segments[pendingProbeType]
	config.Segments[pendingProbeType] = func() config.SegmentWriter { return &pendingProbeWriter{} }
	t.Cleanup(func() {
		if hadPrevious {
			config.Segments[pendingProbeType] = previous
			return
		}

		delete(config.Segments, pendingProbeType)
	})

	configPath := filepath.Join(t.TempDir(), "pending-probe.omp.yaml")
	configYAML := `
daemon_timeout: 20
render_pending_icon: "` + pendingProbeIcon + `"
prompt:
  - segments: ["probe.main"]

probe.main:
  type: "pending_probe"
  template: "READY"
`
	require.NoError(t, os.WriteFile(configPath, []byte(configYAML), 0o644))

	return configPath
}

// drainRenderStream mirrors Server.RenderPrompt: it seeds itself with the
// sequence StartRender reported and pumps NextUpdate until the render
// completes, returning the last bundle the shell would have been left with.
func drainRenderStream(t *testing.T, daemon *Daemon, sessionID, configPath string) PromptBundle {
	t.Helper()

	flags := &runtime.Flags{
		ConfigPath:    configPath,
		Shell:         shell.GENERIC,
		TerminalWidth: 80,
		Plain:         true,
	}

	initial := daemon.StartRender(RenderRequest{
		SessionID: sessionID,
		Flags:     flags,
		Cancel:    CancelHard,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	last := initial.Bundle
	sequence := initial.Sequence
	for {
		update, ok := daemon.NextUpdate(ctx, sessionID, sequence)
		if !ok {
			break
		}

		sequence = update.Sequence
		last = update.Bundle
		if update.Segment == renderCompletePayload {
			break
		}
	}

	require.NoError(t, ctx.Err(), "the render stream never completed; the prompt would stay pending")

	return last
}

// TestRenderStreamAlwaysResolvesPendingPlaceholders is the end-to-end guard on
// the pending-render contract: whenever a render hands the shell a prompt with
// pending placeholders, the shell must eventually receive a replacement without
// them, through the same StartRender/NextUpdate path the gRPC handler drives.
//
// Both ways this used to break were timing races around a segment finishing
// next to the daemon timeout, so the segment's cost is swept across that
// boundary. The window is narrow enough that this sweep is a contract guard
// rather than a reproduction — the two races are pinned deterministically by
// TestStartRenderStreamsFromBeforeItsOwnUpdates and by PrimaryStreaming now
// reporting its pending state instead of the pipeline re-deriving it:
//
//   - RenderPipeline.Start asked the engine for its pending count after
//     PrimaryStreaming returned. The publisher goroutine draining the last
//     result in between made that count read 0, so the pipeline declared the
//     render finished and returned the placeholder prompt as final, with no
//     stream left to correct it.
//   - Daemon.StartRender read the client's starting sequence from the hub after
//     the render had already begun publishing, so updates published while it ran
//     were skipped. When the completion landed there, NextUpdate waited on a
//     sequence that never arrived.
func TestRenderStreamAlwaysResolvesPendingPlaceholders(t *testing.T) {
	configPath := pendingProbeConfig(t)

	// Sweep the segment cost across the daemon timeout in sub-millisecond
	// steps so completions land on both sides of it, and repeatedly inside the
	// narrow window between the initial render and the pipeline's decision.
	const steps = 120
	for step := range steps {
		delay := pendingProbeTimeout - 3*time.Millisecond + time.Duration(step%60)*100*time.Microsecond
		pendingProbeDelay.Store(int64(delay))

		daemon := newRenderDaemon(NewEngineRegistry(prompt.New), nil)
		bundle := drainRenderStream(t, daemon, "session-a", configPath)

		require.NotContains(t, bundle.Primary, pendingProbeIcon,
			"step %d (segment cost %s): the shell was left showing a pending placeholder", step, delay)
		require.Contains(t, bundle.Primary, "READY",
			"step %d (segment cost %s): the resolved segment text never reached the shell", step, delay)

		daemon.CompleteSession("session-a")
	}
}

// TestStartRenderStreamsFromBeforeItsOwnUpdates pins the sequence baseline
// directly: a render must report a starting sequence taken before it could
// publish, so a client streaming from it cannot step over its own generation's
// updates. Reading the hub after the render instead makes the completion
// unreachable, and the client waits forever on a prompt full of placeholders.
func TestStartRenderStreamsFromBeforeItsOwnUpdates(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	renderer := &hubPublishingRenderer{}
	daemon := newRenderDaemon(registry, renderer)
	sessionID := "session-a"

	// Publish from inside the render, exactly where the streaming publisher
	// goroutine does, and with the render's own ID so the relay accepts it.
	renderer.publish = func() {
		_, renderID, ok := registry.GetActiveRender(sessionID)
		require.True(t, ok)
		daemon.SessionHub(sessionID).Publish(renderCompletePayload, renderID)
	}

	initial := daemon.StartRender(RenderRequest{
		SessionID: sessionID,
		Flags:     &runtime.Flags{},
		Cancel:    CancelHard,
	})
	renderer.publish = nil

	require.Equal(t, uint64(0), initial.Sequence,
		"the render published its completion while it ran; a baseline read afterwards skips it")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	update, ok := daemon.NextUpdate(ctx, sessionID, initial.Sequence)
	require.True(t, ok, "the completion published during the render never reached the client")
	require.Equal(t, renderCompletePayload, update.Segment)
}

// TestRepaintReleasesTheSupersededRenderGate covers the reload-gate accounting
// a soft cancel used to drop. A repaint replaces the session's render handle
// without completing it, so the retired handle's gate slot has to be returned
// explicitly — otherwise every vim-mode toggle leaks one, and the first config
// reload afterwards blocks forever with the gate marked reloading, which
// deadlocks every later render.
func TestRepaintReleasesTheSupersededRenderGate(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	daemon := newRenderDaemon(registry, &rendererStub{})
	sessionID := "session-a"

	daemon.StartRender(RenderRequest{SessionID: sessionID, Flags: &runtime.Flags{}, Cancel: CancelHard})

	for range 5 {
		daemon.StartRender(RenderRequest{
			SessionID: sessionID,
			Flags:     &runtime.Flags{VimMode: "normal"},
			Cancel:    CancelSoft,
		})
	}

	active, _ := daemon.Snapshot()
	require.Equal(t, 1, active, "each repaint leaked the superseded handle's reload-gate slot")

	daemon.CompleteSession(sessionID)

	active, _ = daemon.Snapshot()
	require.Equal(t, 0, active)

	reloaded := make(chan struct{})
	go func() {
		daemon.Reload(func() {})
		close(reloaded)
	}()

	select {
	case <-reloaded:
	case <-time.After(2 * time.Second):
		t.Fatal("config reload blocked on leaked reload-gate slots")
	}
}

// hubPublishingRenderer runs publish while the pipeline is rendering, standing
// in for the streaming publisher goroutine at a point the test controls.
type hubPublishingRenderer struct {
	publish func()
}

func (renderer *hubPublishingRenderer) Bundle(_ *prompt.Engine, primary string, _ bundleOptions) PromptBundle {
	if renderer.publish != nil {
		renderer.publish()
	}

	return PromptBundle{Primary: primary}
}
