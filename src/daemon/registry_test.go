package daemon

import (
	"context"
	"testing"

	"github.com/jandedobbeleer/oh-my-posh/src/prompt"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime"

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

	registry.SetActiveRenderCancel("session-a", cancelA)
	registry.SetActiveRenderCancel("session-b", cancelB)
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
