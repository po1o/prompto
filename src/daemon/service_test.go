package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/jandedobbeleer/oh-my-posh/src/prompt"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime"

	"github.com/stretchr/testify/require"
)

func TestServiceStartRenderReturnsInitialBundle(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	renderer := &rendererStub{}
	service := NewService(registry, nil, renderer)

	response := service.StartRender(RenderRequest{
		SessionID: "session-a",
		Flags:     &runtime.Flags{},
	})

	require.Equal(t, "initial", response.Type)
	require.Equal(t, "render", response.Bundle.Primary)
	require.Equal(t, 1, service.SessionCount())

	service.CompleteSession("session-a")
}

func TestServiceStartRenderReplacesExistingSessionHandle(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	renderer := &rendererStub{}
	service := NewService(registry, nil, renderer)

	service.StartRender(RenderRequest{SessionID: "session-a", Flags: &runtime.Flags{}})
	active, _ := service.Snapshot()
	require.Equal(t, 1, active)

	service.StartRender(RenderRequest{SessionID: "session-a", Flags: &runtime.Flags{}})
	active, _ = service.Snapshot()
	require.Equal(t, 1, active)
	require.Equal(t, 1, service.SessionCount())

	service.CompleteSession("session-a")
}

func TestServiceNextUpdateReturnsUpdateResponse(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	renderer := &rendererStub{}
	service := NewService(registry, nil, renderer)

	service.StartRender(RenderRequest{SessionID: "session-a", Flags: &runtime.Flags{}})
	go func() {
		time.Sleep(20 * time.Millisecond)
		service.SessionHub("session-a").Publish("path.main")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	response, ok := service.NextUpdate(ctx, "session-a", 0)
	require.True(t, ok)
	require.Equal(t, "update", response.Type)
	require.Equal(t, uint64(1), response.Sequence)
	require.Equal(t, "path.main", response.Segment)
	require.Equal(t, "render", response.Bundle.Primary)

	service.CompleteSession("session-a")
}

func TestServiceNextUpdateReturnsFalseForUnknownSession(t *testing.T) {
	service := NewService(NewEngineRegistry(nil), nil, &rendererStub{})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, ok := service.NextUpdate(ctx, "missing", 0)
	require.False(t, ok)
}

func TestServiceCompleteSessionRemovesActiveHandle(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	service := NewService(registry, nil, &rendererStub{})
	service.StartRender(RenderRequest{SessionID: "session-a", Flags: &runtime.Flags{}})
	require.Equal(t, 1, service.SessionCount())

	service.CompleteSession("session-a")
	require.Equal(t, 0, service.SessionCount())

	active, reloading := service.Snapshot()
	require.Equal(t, 0, active)
	require.False(t, reloading)
}
