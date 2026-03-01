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
