package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStreamRelayNextReplaysLatestWhenBehind(t *testing.T) {
	hub := NewSessionUpdateHub()
	hub.Publish("first")
	hub.Publish("second")

	relay := NewStreamRelay(hub)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	snapshot, ok := relay.Next(ctx, 1, 0)
	require.True(t, ok)
	require.Equal(t, uint64(2), snapshot.Sequence)
	require.Equal(t, "second", snapshot.Payload)
}

func TestStreamRelayNextWaitsForNewUpdate(t *testing.T) {
	hub := NewSessionUpdateHub()
	relay := NewStreamRelay(hub)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	result := make(chan UpdateSnapshot, 1)
	done := make(chan bool, 1)
	go func() {
		snapshot, ok := relay.Next(ctx, 0, 0)
		if ok {
			result <- snapshot
		}
		done <- ok
	}()

	time.Sleep(20 * time.Millisecond)
	hub.Publish("first")

	select {
	case ok := <-done:
		require.True(t, ok)
	case <-time.After(250 * time.Millisecond):
		t.Fatal("relay should receive published update")
	}

	select {
	case snapshot := <-result:
		require.Equal(t, uint64(1), snapshot.Sequence)
		require.Equal(t, "first", snapshot.Payload)
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected relay snapshot")
	}
}

func TestStreamRelayNextReturnsFalseOnContextCancel(t *testing.T) {
	hub := NewSessionUpdateHub()
	relay := NewStreamRelay(hub)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, ok := relay.Next(ctx, 0, 0)
	require.False(t, ok)
}

func TestStreamRelayNextHandlesNilRelayOrHub(t *testing.T) {
	var relay *StreamRelay
	_, ok := relay.Next(context.Background(), 0, 0)
	require.False(t, ok)

	relay = NewStreamRelay(nil)
	_, ok = relay.Next(context.Background(), 0, 0)
	require.False(t, ok)
}
