package daemon

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSessionUpdateHubSubscribeReceivesPublishedUpdate(t *testing.T) {
	hub := NewSessionUpdateHub()
	subscriber := hub.Subscribe(0)

	go hub.Publish("payload-1")

	select {
	case snapshot := <-subscriber:
		require.Equal(t, uint64(1), snapshot.Sequence)
		require.Equal(t, "payload-1", snapshot.Payload)
	case <-time.After(250 * time.Millisecond):
		t.Fatal("subscriber should receive update")
	}
}

func TestSessionUpdateHubSubscribeAfterOldSequenceGetsImmediateReplay(t *testing.T) {
	hub := NewSessionUpdateHub()
	hub.Publish("payload-1")
	hub.Publish("payload-2")

	subscriber := hub.Subscribe(1)
	select {
	case snapshot := <-subscriber:
		require.Equal(t, uint64(2), snapshot.Sequence)
		require.Equal(t, "payload-2", snapshot.Payload)
	case <-time.After(250 * time.Millisecond):
		t.Fatal("subscriber should immediately replay latest snapshot")
	}
}

func TestSessionUpdateHubLastReturnsCurrentSnapshot(t *testing.T) {
	hub := NewSessionUpdateHub()

	_, ok := hub.Last()
	require.False(t, ok)

	hub.Publish("payload-1")
	snapshot, ok := hub.Last()
	require.True(t, ok)
	require.Equal(t, uint64(1), snapshot.Sequence)
	require.Equal(t, "payload-1", snapshot.Payload)
}

func TestSessionUpdateHubPublishNotifiesAllPendingSubscribers(t *testing.T) {
	hub := NewSessionUpdateHub()
	first := hub.Subscribe(0)
	second := hub.Subscribe(0)

	go hub.Publish("payload-1")

	for _, subscriber := range []<-chan UpdateSnapshot{first, second} {
		select {
		case snapshot := <-subscriber:
			require.Equal(t, uint64(1), snapshot.Sequence)
			require.Equal(t, "payload-1", snapshot.Payload)
		case <-time.After(250 * time.Millisecond):
			t.Fatal("subscriber should receive shared published update")
		}
	}
}

func TestSessionUpdateHubConcurrentPublishIncrementsSequence(t *testing.T) {
	hub := NewSessionUpdateHub()
	var wg sync.WaitGroup
	count := 200

	for range count {
		wg.Go(func() {
			hub.Publish("payload")
		})
	}

	wg.Wait()

	snapshot, ok := hub.Last()
	require.True(t, ok)
	require.Equal(t, uint64(count), snapshot.Sequence)
}

// TestSessionUpdateHubBoundsItsHistory pins the catch-up window. The history
// used to grow for the life of the shell, and every subscribe scanned all of
// it. Trimming keeps both bounded; a subscriber inside the window still gets
// the exact successor, and one that has fallen outside resumes at the oldest
// entry held rather than being told there is nothing to read.
func TestSessionUpdateHubBoundsItsHistory(t *testing.T) {
	hub := NewSessionUpdateHub()

	const published = 5 * historyLimit
	for range published {
		hub.Publish("payload")
	}

	hub.mu.Lock()
	retained := len(hub.updates)
	hub.mu.Unlock()
	require.LessOrEqual(t, retained, 2*historyLimit, "history must stay bounded")

	// Inside the window: the exact successor.
	snapshot, ok := hub.nextAfter(published - 2)
	require.True(t, ok)
	require.Equal(t, uint64(published-1), snapshot.Sequence)

	// Outside the window: the oldest entry still held, never "nothing".
	snapshot, ok = hub.nextAfter(0)
	require.True(t, ok)
	require.Equal(t, uint64(published-retained+1), snapshot.Sequence)

	// Caught up: nothing to read.
	_, ok = hub.nextAfter(published)
	require.False(t, ok)
}

func (hub *SessionUpdateHub) nextAfter(after uint64) (UpdateSnapshot, bool) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	return hub.nextAfterLocked(after)
}
