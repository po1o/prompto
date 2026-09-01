package daemon

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/po1o/prompto/src/prompt"
	"github.com/po1o/prompto/src/runtime"
	"github.com/po1o/prompto/src/shell"

	"github.com/stretchr/testify/require"
)

func TestGetOrCreateEngineReturnsSameEnginePerSession(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})

	engineA := registry.GetOrCreateEngine("session-a", &runtime.Flags{})
	engineA2 := registry.GetOrCreateEngine("session-a", &runtime.Flags{})
	engineB := registry.GetOrCreateEngine("session-b", &runtime.Flags{})

	require.Same(t, engineA, engineA2)
	require.NotSame(t, engineA, engineB)
}

func TestCancelActiveRenderCancelsOnlyCurrentSession(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})

	registry.GetOrCreateEngine("session-a", &runtime.Flags{})
	registry.GetOrCreateEngine("session-b", &runtime.Flags{})

	ctxA, cancelA := context.WithCancel(context.Background())
	ctxB, cancelB := context.WithCancel(context.Background())
	t.Cleanup(cancelA)
	t.Cleanup(cancelB)

	registry.SetActiveRender("session-a", ctxA, cancelA)
	registry.SetActiveRender("session-b", ctxB, cancelB)
	registry.CancelActiveRender("session-a")

	select {
	case <-ctxA.Done():
	default:
		t.Fatal("session-a render should have been canceled")
	}

	select {
	case <-ctxB.Done():
		t.Fatal("session-b render should not have been canceled")
	default:
	}
}

func TestRemoveSessionCreatesNewEngineOnNextRequest(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})

	first := registry.GetOrCreateEngine("session-a", &runtime.Flags{})
	registry.RemoveSession("session-a")
	second := registry.GetOrCreateEngine("session-a", &runtime.Flags{})

	require.NotSame(t, first, second)
}

func TestStartRenderSoftCancelReattachesToActiveContext(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})

	first := registry.StartRender("session-a", &runtime.Flags{}, CancelHard)
	second := registry.StartRender("session-a", &runtime.Flags{}, CancelSoft)

	require.NotNil(t, first.Context)
	require.NotNil(t, second.Context)
	require.True(t, second.Reattached)
	require.Same(t, first.Engine, second.Engine)
	require.Same(t, first.Context, second.Context)
}

func TestStartRenderHardCancelCancelsActiveRender(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})

	first := registry.StartRender("session-a", &runtime.Flags{}, CancelHard)
	second := registry.StartRender("session-a", &runtime.Flags{}, CancelHard)

	require.False(t, second.Reattached)
	require.NotSame(t, first.Context, second.Context)

	select {
	case <-first.Context.Done():
	default:
		t.Fatal("first render context should be canceled by a hard cancel")
	}

	select {
	case <-second.Context.Done():
		t.Fatal("second render context should stay active")
	default:
	}
}

func TestRenderHandleCompleteClearsOnlyMatchingActiveRender(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})

	first := registry.StartRender("session-a", &runtime.Flags{}, CancelHard)
	second := registry.StartRender("session-a", &runtime.Flags{}, CancelHard)

	first.Complete()

	activeContext, _, ok := registry.GetActiveRender("session-a")
	require.True(t, ok)
	require.Same(t, second.Context, activeContext)

	second.Complete()

	_, _, ok = registry.GetActiveRender("session-a")
	require.False(t, ok)
}

// A session is keyed by the client's pid and the OS reuses pids, so a new shell
// can land on a dead one's registry entry. Nothing else reconciles the two —
// applyRenderFlags refreshes flags and env but never the config — so without
// this the first config a session rendered would keep rendering for the life of
// the daemon: a console config answering a desktop session, or the reverse.
func TestGetOrCreateEngineRebuildsWhenTheConfigPathChanges(t *testing.T) {
	dir := t.TempDir()
	body := "prompt:\n  - segments: [\"path\"]\n\npath:\n  type: \"text\"\n  template: \" ~ \"\n"

	gui := filepath.Join(dir, "config.yaml")
	console := filepath.Join(dir, "config.console.yaml")
	require.NoError(t, os.WriteFile(gui, []byte(body), 0o644))
	require.NoError(t, os.WriteFile(console, []byte(body), 0o644))

	registry := NewEngineRegistry(prompt.New)
	flagsFor := func(path string) *runtime.Flags {
		return &runtime.Flags{ConfigPath: path, Shell: shell.GENERIC, TerminalWidth: 80}
	}

	first := registry.GetOrCreateEngine("session", flagsFor(gui))
	require.NotNil(t, first.LayoutConfig)
	require.Equal(t, gui, first.LayoutConfig.Source)

	second := registry.GetOrCreateEngine("session", flagsFor(console))
	require.Equal(t, console, second.LayoutConfig.Source, "the request's config must win over the cached engine")
	require.True(t, second.LayoutConfig.Console, "the console variant should be recognised")

	// Asking again for the same config reuses the engine, so per-session caches
	// survive an ordinary render.
	again := registry.GetOrCreateEngine("session", flagsFor(console))
	require.Same(t, second, again)
}
