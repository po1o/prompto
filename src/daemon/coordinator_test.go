package daemon

import (
	"testing"

	"github.com/jandedobbeleer/oh-my-posh/src/prompt"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime"

	"github.com/stretchr/testify/require"
)

func TestStartRenderRepaintReattachesToActiveContext(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	coordinator := NewRenderCoordinator(registry)

	first := coordinator.StartRender("session-a", &runtime.Flags{}, false)
	second := coordinator.StartRender("session-a", &runtime.Flags{}, true)

	require.NotNil(t, first.Context)
	require.NotNil(t, second.Context)
	require.True(t, second.Reattached)
	require.Same(t, first.Engine, second.Engine)
	require.Same(t, first.Context, second.Context)
}

func TestStartRenderNonRepaintCancelsActiveRender(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	coordinator := NewRenderCoordinator(registry)

	first := coordinator.StartRender("session-a", &runtime.Flags{}, false)
	second := coordinator.StartRender("session-a", &runtime.Flags{}, false)

	require.False(t, second.Reattached)
	require.NotSame(t, first.Context, second.Context)

	select {
	case <-first.Context.Done():
	default:
		t.Fatal("first render context should be canceled by non-repaint start")
	}

	select {
	case <-second.Context.Done():
		t.Fatal("second render context should stay active")
	default:
	}
}

func TestCompleteClearsOnlyMatchingActiveRender(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	coordinator := NewRenderCoordinator(registry)

	first := coordinator.StartRender("session-a", &runtime.Flags{}, false)
	second := coordinator.StartRender("session-a", &runtime.Flags{}, false)

	first.Complete()

	activeContext, ok := registry.GetActiveRenderContext("session-a")
	require.True(t, ok)
	require.Same(t, second.Context, activeContext)

	second.Complete()

	_, ok = registry.GetActiveRenderContext("session-a")
	require.False(t, ok)
}
