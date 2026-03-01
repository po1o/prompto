package daemon

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type callbackSetterStub struct {
	callback func(string)
}

func (stub *callbackSetterStub) SetUpdateCallback(callback func(string)) {
	stub.callback = callback
}

func TestBindSegmentUpdatesPublishesToSessionHub(t *testing.T) {
	store := NewPromptSessionStore(nil)
	engine := &callbackSetterStub{}

	BindSegmentUpdates("session-a", engine, store)
	require.NotNil(t, engine.callback)

	wait := store.Hub("session-a").Subscribe(0)
	engine.callback("path.main")

	select {
	case update := <-wait:
		require.Equal(t, uint64(1), update.Sequence)
		require.Equal(t, "path.main", update.Payload)
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected session update to be published")
	}
}

func TestBindSegmentUpdatesHandlesNilInputs(t *testing.T) {
	BindSegmentUpdates("session-a", nil, NewPromptSessionStore(nil))
	BindSegmentUpdates("session-a", &callbackSetterStub{}, nil)
}

func TestClearSegmentUpdatesResetsCallback(t *testing.T) {
	engine := &callbackSetterStub{}
	engine.SetUpdateCallback(func(string) {})
	require.NotNil(t, engine.callback)

	ClearSegmentUpdates(engine)
	require.Nil(t, engine.callback)
}

func TestClearSegmentUpdatesHandlesNilEngine(t *testing.T) {
	ClearSegmentUpdates(nil)
}
