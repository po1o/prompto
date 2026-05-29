package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/po1o/prompto/src/prompt"
	"github.com/po1o/prompto/src/runtime"

	"github.com/stretchr/testify/require"
)

// Characterization tests (Task B2) for the three canonical vim-mode
// scenarios from .claude/docs/daemon-vim-mode-plan.md. They assert the
// OBSERVABLE cancel invariants — context preservation, context abort, and
// single-computation reuse — not internal types. They must pass on the
// current tree and keep passing through the C-series refactor (especially
// C1's CancelKind introduction, which changes how the cancel decision is
// represented but not what it does).
//
// When C2 merges Service into Daemon, only newScenarioHarness changes; the
// three Test_Scenario_* bodies keep their assertions verbatim.

type scenarioHarness struct {
	service  *Service
	registry *EngineRegistry
}

func newScenarioHarness(t *testing.T) *scenarioHarness {
	t.Helper()

	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	service := NewService(registry, nil, &rendererStub{})

	return &scenarioHarness{service: service, registry: registry}
}

// startCommand issues a normal (non-repaint) render — a Hard cancel of any
// prior in-flight render for the session.
func (h *scenarioHarness) startCommand(sessionID string) {
	h.service.StartRender(RenderRequest{
		SessionID: sessionID,
		Flags:     &runtime.Flags{},
		Repaint:   false,
	})
}

// toggleVimMode issues a repaint render — a Soft cancel that preserves the
// in-flight computation for the session.
func (h *scenarioHarness) toggleVimMode(sessionID, mode string) {
	h.service.StartRender(RenderRequest{
		SessionID: sessionID,
		Flags:     &runtime.Flags{VimMode: mode},
		Repaint:   true,
	})
}

// Scenario 1: toggling vim mode preserves the in-flight computation
// (Soft cancel). The render context and render ID are reused, the context
// is NOT aborted, and an update published on the preserved render reaches
// the new subscriber — i.e. the heavy segment runs once and its result is
// reused.
func Test_Scenario_SoftCancel_VimToggle(t *testing.T) {
	h := newScenarioHarness(t)
	const sessionID = "session-soft"

	h.startCommand(sessionID)
	firstCtx, firstID, ok := h.registry.GetActiveRender(sessionID)
	require.True(t, ok, "a render must be active after the initial command")

	h.toggleVimMode(sessionID, "normal")
	secondCtx, secondID, ok := h.registry.GetActiveRender(sessionID)
	require.True(t, ok, "a render must still be active after the vim toggle")

	// The computation is preserved, not restarted.
	require.Same(t, firstCtx, secondCtx, "soft cancel must reuse the same render context")
	require.Equal(t, firstID, secondID, "soft cancel must reuse the same render ID")
	require.NoError(t, firstCtx.Err(), "soft cancel must NOT abort the in-flight computation")

	// The preserved computation's result reaches the post-toggle subscriber.
	go func() {
		time.Sleep(10 * time.Millisecond)
		h.service.SessionHub(sessionID).Publish("segment.git", firstID)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	update, ok := h.service.NextUpdate(ctx, sessionID, 0)
	require.True(t, ok, "the preserved computation's update must stream to the new subscriber")
	require.Equal(t, "segment.git", update.Segment)

	h.service.CompleteSession(sessionID)
}

// Scenario 2: running a new command aborts the in-flight computation
// (Hard cancel). The prior render context is cancelled — this is the gate
// that prevents a stale computation from writing outdated data to the
// cache — and a fresh render context with a new ID replaces it.
func Test_Scenario_HardCancel_NewCommand(t *testing.T) {
	h := newScenarioHarness(t)
	const sessionID = "session-hard"

	h.startCommand(sessionID)
	firstCtx, firstID, ok := h.registry.GetActiveRender(sessionID)
	require.True(t, ok, "a render must be active after the first command")

	h.startCommand(sessionID)
	secondCtx, secondID, ok := h.registry.GetActiveRender(sessionID)
	require.True(t, ok, "a render must be active after the second command")

	// The prior computation is aborted: ctx.Err() != nil is exactly the
	// condition that cache-write code checks before persisting a segment.
	require.Error(t, firstCtx.Err(), "hard cancel MUST abort the prior render context (cache-pollution safety)")

	// A fresh computation replaces it.
	require.NotSame(t, firstCtx, secondCtx, "hard cancel must start a new render context")
	require.NotEqual(t, firstID, secondID, "hard cancel must allocate a new render ID")
	require.NoError(t, secondCtx.Err(), "the replacement render context must be live")

	h.service.CompleteSession(sessionID)
}

// Scenario 3: rapid-fire toggles never restart the computation. Across many
// repaints the render context and ID stay identical and the context is
// never aborted, so the heavy segment runs exactly once. A single update on
// the original render still reaches whichever subscriber is current.
func Test_Scenario_RapidFireToggles_RunsOnce(t *testing.T) {
	h := newScenarioHarness(t)
	const sessionID = "session-rapid"

	h.startCommand(sessionID)
	baseCtx, baseID, ok := h.registry.GetActiveRender(sessionID)
	require.True(t, ok, "a render must be active after the initial command")

	modes := []string{"normal", "insert"}
	for i := range 200 {
		h.toggleVimMode(sessionID, modes[i%2])

		ctx, id, exists := h.registry.GetActiveRender(sessionID)
		require.True(t, exists, "render must stay active across toggle %d", i)
		require.Same(t, baseCtx, ctx, "toggle %d must reuse the original render context", i)
		require.Equal(t, baseID, id, "toggle %d must reuse the original render ID", i)
		require.NoError(t, ctx.Err(), "toggle %d must not abort the computation", i)
	}

	// The single original computation still delivers to the current subscriber.
	go func() {
		time.Sleep(10 * time.Millisecond)
		h.service.SessionHub(sessionID).Publish("segment.git", baseID)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	update, ok := h.service.NextUpdate(ctx, sessionID, 0)
	require.True(t, ok, "the single computation's update must reach the final subscriber")
	require.Equal(t, "segment.git", update.Segment)

	h.service.CompleteSession(sessionID)
}
