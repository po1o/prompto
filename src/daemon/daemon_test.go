package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/jandedobbeleer/oh-my-posh/src/runtime"

	"github.com/stretchr/testify/require"
)

func TestDaemonStartRenderAndNextUpdateFlow(t *testing.T) {
	daemon := New(&rendererStub{})

	initial := daemon.StartRender(RenderRequest{
		SessionID: "session-a",
		Flags:     &runtime.Flags{},
	})
	require.Equal(t, "initial", initial.Type)
	require.Equal(t, "render", initial.Bundle.Primary)

	go func() {
		time.Sleep(20 * time.Millisecond)
		daemon.SessionHub("session-a").Publish("path.main")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	update, ok := daemon.NextUpdate(ctx, "session-a", 0)
	require.True(t, ok)
	require.Equal(t, "update", update.Type)
	require.Equal(t, uint64(1), update.Sequence)
	require.Equal(t, "path.main", update.Segment)
}

func TestDaemonReloadBlocksNewRenderRequests(t *testing.T) {
	daemon := New(&rendererStub{})

	reloadStarted := make(chan struct{})
	reloadDone := make(chan struct{})
	go func() {
		daemon.Reload(func() {
			close(reloadStarted)
			time.Sleep(120 * time.Millisecond)
		})
		close(reloadDone)
	}()

	select {
	case <-reloadStarted:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("reload should start")
	}

	renderDone := make(chan struct{})
	go func() {
		_ = daemon.StartRender(RenderRequest{
			SessionID: "session-a",
			Flags:     &runtime.Flags{},
		})
		close(renderDone)
	}()

	select {
	case <-renderDone:
		t.Fatal("render should be blocked while reload is active")
	case <-time.After(50 * time.Millisecond):
	}

	select {
	case <-reloadDone:
	case <-time.After(time.Second):
		t.Fatal("reload should complete")
	}

	select {
	case <-renderDone:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("render should proceed after reload")
	}
}

func TestDaemonStopPreventsNewOperations(t *testing.T) {
	daemon := New(&rendererStub{})
	daemon.Stop()

	initial := daemon.StartRender(RenderRequest{
		SessionID: "session-a",
		Flags:     &runtime.Flags{},
	})
	require.Equal(t, "stopped", initial.Type)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, ok := daemon.NextUpdate(ctx, "session-a", 0)
	require.False(t, ok)
}

func TestDaemonAutoStopsAfterIdleTimeoutWithoutSessions(t *testing.T) {
	daemon := NewWithIdleTimeout(25*time.Millisecond, &rendererStub{})

	time.Sleep(60 * time.Millisecond)

	response := daemon.StartRender(RenderRequest{
		SessionID: "session-a",
		Flags:     &runtime.Flags{},
	})
	require.Equal(t, "stopped", response.Type)
}

func TestDaemonIdleTimerStartsAfterSessionCompletion(t *testing.T) {
	daemon := NewWithIdleTimeout(30*time.Millisecond, &rendererStub{})

	initial := daemon.StartRender(RenderRequest{
		SessionID: "session-a",
		Flags:     &runtime.Flags{},
	})
	require.Equal(t, "initial", initial.Type)

	time.Sleep(50 * time.Millisecond)
	stillRunning := daemon.StartRender(RenderRequest{
		SessionID: "session-b",
		Flags:     &runtime.Flags{},
	})
	require.Equal(t, "initial", stillRunning.Type)

	daemon.CompleteSession("session-a")
	daemon.CompleteSession("session-b")
	time.Sleep(70 * time.Millisecond)

	stopped := daemon.StartRender(RenderRequest{
		SessionID: "session-c",
		Flags:     &runtime.Flags{},
	})
	require.Equal(t, "stopped", stopped.Type)
}
