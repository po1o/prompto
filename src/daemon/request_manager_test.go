package daemon

import (
	"testing"
	"time"

	"github.com/jandedobbeleer/oh-my-posh/src/prompt"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime"

	"github.com/stretchr/testify/require"
)

func TestStartRequestTracksActiveUntilComplete(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	manager := NewRequestManager(registry, nil)

	handle := manager.StartRequest("session-a", &runtime.Flags{}, false)
	active, reloading := manager.Snapshot()
	require.Equal(t, 1, active)
	require.False(t, reloading)
	require.NotNil(t, handle.Render)

	handle.Complete()
	active, reloading = manager.Snapshot()
	require.Equal(t, 0, active)
	require.False(t, reloading)
}

func TestReloadWaitsForActiveRequests(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	manager := NewRequestManager(registry, nil)

	handle := manager.StartRequest("session-a", &runtime.Flags{}, false)
	defer handle.Complete()

	reloadDone := make(chan struct{})
	go func() {
		manager.Reload(nil)
		close(reloadDone)
	}()

	requireReloadingState(t, manager, true)

	select {
	case <-reloadDone:
		t.Fatal("reload should wait for active request completion")
	case <-time.After(50 * time.Millisecond):
	}

	handle.Complete()

	select {
	case <-reloadDone:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("reload should finish after active request completes")
	}
}

func TestRequestManagerStartRequestBlocksWhileReloading(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	manager := NewRequestManager(registry, nil)

	reloadDone := make(chan struct{})
	go func() {
		manager.Reload(func() {
			time.Sleep(100 * time.Millisecond)
		})
		close(reloadDone)
	}()

	requireReloadingState(t, manager, true)

	requestStarted := make(chan struct{})
	var handle *RequestHandle
	go func() {
		handle = manager.StartRequest("session-a", &runtime.Flags{}, false)
		close(requestStarted)
	}()

	select {
	case <-requestStarted:
		t.Fatal("request should block while reload is active")
	case <-time.After(50 * time.Millisecond):
	}

	select {
	case <-reloadDone:
	case <-time.After(350 * time.Millisecond):
		t.Fatal("reload should finish")
	}

	select {
	case <-requestStarted:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("request should continue once reload ends")
	}

	handle.Complete()
}

func TestStartRequestReusesEngineAndReattachesOnRepaint(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	manager := NewRequestManager(registry, nil)

	first := manager.StartRequest("session-a", &runtime.Flags{}, false)
	repaint := manager.StartRequest("session-a", &runtime.Flags{}, true)

	require.Same(t, first.Render.Engine, repaint.Render.Engine)
	require.True(t, repaint.Render.Reattached)
	require.Same(t, first.Render.Context, repaint.Render.Context)

	first.Complete()
	repaint.Complete()
}

func TestStartRequestNonRepaintCancelsPreviousContext(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	manager := NewRequestManager(registry, nil)

	first := manager.StartRequest("session-a", &runtime.Flags{}, false)
	second := manager.StartRequest("session-a", &runtime.Flags{}, false)

	require.False(t, second.Render.Reattached)
	require.NotSame(t, first.Render.Context, second.Render.Context)

	select {
	case <-first.Render.Context.Done():
	default:
		t.Fatal("first context should be canceled by non-repaint request")
	}

	second.Complete()
	first.Complete()
}

func requireReloadingState(t *testing.T, manager *RequestManager, expected bool) {
	t.Helper()

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		_, reloading := manager.Snapshot()
		if reloading == expected {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}

	_, reloading := manager.Snapshot()
	t.Fatalf("expected reloading=%t, got %t", expected, reloading)
}
