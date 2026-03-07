package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/po1o/prompto/src/prompt"
	"github.com/po1o/prompto/src/runtime"

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

func TestServiceStartRenderRepaintReattachesActiveRender(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	service := NewService(registry, nil, &rendererStub{})

	service.StartRender(RenderRequest{SessionID: "session-a", Flags: &runtime.Flags{}, Repaint: false})
	firstContext, firstID, ok := registry.GetActiveRender("session-a")
	require.True(t, ok)

	service.StartRender(RenderRequest{SessionID: "session-a", Flags: &runtime.Flags{VimMode: "normal"}, Repaint: true})
	secondContext, secondID, ok := registry.GetActiveRender("session-a")
	require.True(t, ok)

	require.Same(t, firstContext, secondContext)
	require.Equal(t, firstID, secondID)
}

func TestServiceRepaintKeepsContextAndStreamsPendingUpdates(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	service := NewService(registry, nil, &rendererStub{})
	sessionID := sessionIDFixture

	service.StartRender(RenderRequest{SessionID: sessionID, Flags: &runtime.Flags{}, Repaint: false})
	firstContext, renderID, ok := registry.GetActiveRender(sessionID)
	require.True(t, ok)

	service.StartRender(RenderRequest{SessionID: sessionID, Flags: &runtime.Flags{VimMode: "normal"}, Repaint: true})
	secondContext, secondID, ok := registry.GetActiveRender(sessionID)
	require.True(t, ok)
	require.Same(t, firstContext, secondContext)
	require.Equal(t, renderID, secondID)

	go func() {
		time.Sleep(10 * time.Millisecond)
		service.SessionHub(sessionID).Publish("path.main", renderID)
		service.SessionHub(sessionID).Publish(renderCompletePayload, renderID)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	update, ok := service.NextUpdate(ctx, sessionID, 0)
	require.True(t, ok)
	require.Equal(t, "path.main", update.Segment)
}

func TestServiceRapidRepaintDoesNotCreateNewRenderContext(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	service := NewService(registry, nil, &rendererStub{})
	sessionID := sessionIDFixture

	service.StartRender(RenderRequest{SessionID: sessionID, Flags: &runtime.Flags{}, Repaint: false})
	baseContext, baseID, ok := registry.GetActiveRender(sessionID)
	require.True(t, ok)

	for i := range 200 {
		service.StartRender(RenderRequest{
			SessionID: sessionID,
			Flags:     &runtime.Flags{VimMode: "normal"},
			Repaint:   true,
		})

		ctx, renderID, exists := registry.GetActiveRender(sessionID)
		require.True(t, exists)
		require.Same(t, baseContext, ctx)
		require.Equal(t, baseID, renderID)
		require.Nil(t, ctx.Err(), "repaint %d should not cancel active render context", i)
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		service.SessionHub(sessionID).Publish("path.main", baseID)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	update, ok := service.NextUpdate(ctx, sessionID, 0)
	require.True(t, ok)
	require.Equal(t, "path.main", update.Segment)
}

func TestServiceDeregistersRenderWhenCompleteUpdateArrives(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	service := NewService(registry, nil, &rendererStub{})
	sessionID := sessionIDFixture

	service.StartRender(RenderRequest{SessionID: sessionID, Flags: &runtime.Flags{}, Repaint: false})
	_, renderID, ok := registry.GetActiveRender(sessionID)
	require.True(t, ok)
	require.Equal(t, 1, service.SessionCount())

	go func() {
		time.Sleep(10 * time.Millisecond)
		service.SessionHub(sessionID).Publish(renderCompletePayload, renderID)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, ok := service.NextUpdate(ctx, sessionID, 0)
	require.True(t, ok)
	require.Equal(t, renderCompletePayload, response.Segment)

	require.Equal(t, 0, service.SessionCount())
	active, reloading := service.Snapshot()
	require.Equal(t, 0, active)
	require.False(t, reloading)
}

func TestServiceNextUpdateContextCancelKeepsActiveRender(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	service := NewService(registry, nil, &rendererStub{})
	sessionID := sessionIDFixture

	service.StartRender(RenderRequest{SessionID: sessionID, Flags: &runtime.Flags{}, Repaint: false})
	_, renderID, ok := registry.GetActiveRender(sessionID)
	require.True(t, ok)
	require.Equal(t, 1, service.SessionCount())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, ok = service.NextUpdate(ctx, sessionID, 0)
	require.False(t, ok)
	require.Equal(t, 1, service.SessionCount())

	go func() {
		time.Sleep(10 * time.Millisecond)
		service.SessionHub(sessionID).Publish("path.main", renderID)
	}()

	freshCtx, freshCancel := context.WithTimeout(context.Background(), time.Second)
	defer freshCancel()
	update, ok := service.NextUpdate(freshCtx, sessionID, 0)
	require.True(t, ok)
	require.Equal(t, "path.main", update.Segment)
}

func TestServiceNextUpdateStopsWhenRenderIsSuperseded(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	service := NewService(registry, nil, &rendererStub{})
	sessionID := sessionIDFixture

	service.StartRender(RenderRequest{SessionID: sessionID, Flags: &runtime.Flags{}, Repaint: false})

	firstDone := make(chan bool, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, ok := service.NextUpdate(ctx, sessionID, 0)
		firstDone <- ok
	}()

	time.Sleep(20 * time.Millisecond)
	service.StartRender(RenderRequest{SessionID: sessionID, Flags: &runtime.Flags{}, Repaint: false})

	select {
	case ok := <-firstDone:
		require.False(t, ok)
	case <-time.After(250 * time.Millisecond):
		t.Fatal("superseded render stream should stop promptly")
	}
}

func TestServiceStartRenderRepaintSkipsCompletedActiveRender(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	service := NewService(registry, nil, &rendererStub{})
	sessionID := sessionIDFixture

	service.StartRender(RenderRequest{SessionID: sessionID, Flags: &runtime.Flags{}, Repaint: false})
	_, renderID, ok := registry.GetActiveRender(sessionID)
	require.True(t, ok)

	service.SessionHub(sessionID).Publish(renderCompletePayload, renderID)

	response := service.StartRender(RenderRequest{
		SessionID: sessionID,
		Flags:     &runtime.Flags{VimMode: "normal"},
		Repaint:   true,
	})

	require.Equal(t, "initial", response.Type)
	require.Equal(t, uint64(1), response.Sequence)
	require.Equal(t, 0, service.SessionCount())

	_, _, ok = registry.GetActiveRender(sessionID)
	require.False(t, ok)
}

func TestServiceStartRenderReattachStartsAfterCurrentSequence(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	service := NewService(registry, nil, &rendererStub{})
	sessionID := sessionIDFixture

	service.StartRender(RenderRequest{SessionID: sessionID, Flags: &runtime.Flags{}, Repaint: false})
	_, renderID, ok := registry.GetActiveRender(sessionID)
	require.True(t, ok)

	snapshot := service.SessionHub(sessionID).Publish("path.main", renderID)

	response := service.StartRender(RenderRequest{
		SessionID: sessionID,
		Flags:     &runtime.Flags{VimMode: "normal"},
		Repaint:   true,
	})

	require.Equal(t, "initial", response.Type)
	require.Equal(t, snapshot.Sequence, response.Sequence)
}
