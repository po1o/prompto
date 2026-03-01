package daemon

import (
	"testing"
	"time"
)

func TestBeginReloadWaitsForActiveRequests(t *testing.T) {
	gate := NewReloadGate()
	doneRequest := gate.StartRequest()

	reloadDone := make(chan struct{})
	go func() {
		gate.BeginReload()
		close(reloadDone)
	}()

	requireReloading(t, gate, true)

	select {
	case <-reloadDone:
		t.Fatal("reload should wait for active requests")
	case <-time.After(50 * time.Millisecond):
	}

	doneRequest()

	select {
	case <-reloadDone:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("reload should proceed after active requests are done")
	}
}

func TestStartRequestBlocksWhileReloading(t *testing.T) {
	gate := NewReloadGate()
	gate.BeginReload()
	requireReloading(t, gate, true)

	requestStarted := make(chan struct{})
	go func() {
		done := gate.StartRequest()
		defer done()
		close(requestStarted)
	}()

	select {
	case <-requestStarted:
		t.Fatal("request should block while reload is active")
	case <-time.After(50 * time.Millisecond):
	}

	gate.EndReload()

	select {
	case <-requestStarted:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("request should start after reload ends")
	}
}

func TestConcurrentReloadsAreSerialized(t *testing.T) {
	gate := NewReloadGate()
	gate.BeginReload()

	secondReloadStarted := make(chan struct{})
	secondReloadDone := make(chan struct{})
	go func() {
		close(secondReloadStarted)
		gate.BeginReload()
		close(secondReloadDone)
	}()

	select {
	case <-secondReloadStarted:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("second reload goroutine should start")
	}

	select {
	case <-secondReloadDone:
		t.Fatal("second reload should wait for first reload to end")
	case <-time.After(50 * time.Millisecond):
	}

	gate.EndReload()

	select {
	case <-secondReloadDone:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("second reload should proceed after first reload ends")
	}

	gate.EndReload()
}

func requireReloading(t *testing.T, gate *ReloadGate, expected bool) {
	t.Helper()

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		_, reloading := gate.Snapshot()
		if reloading == expected {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}

	_, reloading := gate.Snapshot()
	t.Fatalf("expected reloading=%t, got %t", expected, reloading)
}
