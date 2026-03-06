package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/po1o/prompto/src/prompt"
	"github.com/po1o/prompto/src/runtime"

	"github.com/stretchr/testify/require"
)

func TestSessionRenderRuntimeStartRequestReturnsEngineAndRelay(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	sessionRuntime := NewSessionRenderRuntime(registry, nil)

	handle := sessionRuntime.StartRequest("session-a", &runtime.Flags{}, false)
	require.NotNil(t, handle.Engine())
	require.NotNil(t, handle.Relay())

	handle.Complete()
}

func TestSessionRenderRuntimeRelayReadsSessionHubUpdates(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	sessionRuntime := NewSessionRenderRuntime(registry, nil)

	handle := sessionRuntime.StartRequest("session-a", &runtime.Flags{}, false)
	defer handle.Complete()

	sessionRuntime.SessionHub("session-a").Publish("path.main")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	snapshot, ok := handle.Relay().Next(ctx, 0, handle.RenderID())
	require.True(t, ok)
	require.Equal(t, uint64(1), snapshot.Sequence)
	require.Equal(t, "path.main", snapshot.Payload)
}

func TestSessionRenderRuntimeCompleteReleasesActiveRequest(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	sessionRuntime := NewSessionRenderRuntime(registry, nil)

	handle := sessionRuntime.StartRequest("session-a", &runtime.Flags{}, false)
	active, _ := sessionRuntime.Snapshot()
	require.Equal(t, 1, active)

	handle.Complete()
	active, reloading := sessionRuntime.Snapshot()
	require.Equal(t, 0, active)
	require.False(t, reloading)
}

func TestSessionRenderRuntimeRemoveSessionResetsEngineReuse(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	sessionRuntime := NewSessionRenderRuntime(registry, nil)

	first := sessionRuntime.StartRequest("session-a", &runtime.Flags{}, false)
	firstEngine := first.Engine()
	first.Complete()

	sessionRuntime.RemoveSession("session-a")

	second := sessionRuntime.StartRequest("session-a", &runtime.Flags{}, false)
	secondEngine := second.Engine()
	second.Complete()

	require.NotSame(t, firstEngine, secondEngine)
}
