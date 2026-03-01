package daemon

import (
	"testing"

	"github.com/jandedobbeleer/oh-my-posh/src/prompt"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime"

	"github.com/stretchr/testify/require"
)

func TestPromptSessionStoreHubReusesSessionHub(t *testing.T) {
	store := NewPromptSessionStore(nil)

	first := store.Hub("session-a")
	second := store.Hub("session-a")
	other := store.Hub("session-b")

	require.Same(t, first, second)
	require.NotSame(t, first, other)
	require.Equal(t, 2, store.Count())
}

func TestPromptSessionStoreRemoveSessionRemovesHubAndRegistryState(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	store := NewPromptSessionStore(registry)

	engine := registry.GetOrCreateEngine("session-a", &runtime.Flags{})
	store.Hub("session-a")

	store.RemoveSession("session-a")

	require.Equal(t, 0, store.Count())

	nextEngine := registry.GetOrCreateEngine("session-a", &runtime.Flags{})
	require.NotSame(t, engine, nextEngine)
}

func TestPromptSessionStoreRemoveSessionWithoutRegistry(t *testing.T) {
	store := NewPromptSessionStore(nil)
	store.Hub("session-a")

	store.RemoveSession("session-a")

	require.Equal(t, 0, store.Count())
}
