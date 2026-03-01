package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

type managedDaemonStub struct {
	stopped bool
}

func (stub *managedDaemonStub) Stop() {
	stub.stopped = true
}

func TestRunDaemonActionLifecycle(t *testing.T) {
	created := []*managedDaemonStub{}
	controller := newDaemonController(func() managedDaemon {
		instance := &managedDaemonStub{}
		created = append(created, instance)
		return instance
	})

	buffer := new(bytes.Buffer)

	require.NoError(t, runDaemonAction("status", controller, buffer))
	require.Contains(t, buffer.String(), "daemon is not running")
	buffer.Reset()

	require.NoError(t, runDaemonAction("start", controller, buffer))
	require.Contains(t, buffer.String(), "daemon started")
	require.True(t, controller.Running())
	require.Len(t, created, 1)
	buffer.Reset()

	require.NoError(t, runDaemonAction("start", controller, buffer))
	require.Contains(t, buffer.String(), "daemon is already running")
	require.Len(t, created, 1)
	buffer.Reset()

	require.NoError(t, runDaemonAction("status", controller, buffer))
	require.Contains(t, buffer.String(), "daemon is running")
	buffer.Reset()

	require.NoError(t, runDaemonAction("restart", controller, buffer))
	require.Contains(t, buffer.String(), "daemon restarted")
	require.Len(t, created, 2)
	require.True(t, created[0].stopped)
	require.False(t, created[1].stopped)
	buffer.Reset()

	require.NoError(t, runDaemonAction("stop", controller, buffer))
	require.Contains(t, buffer.String(), "daemon stopped")
	require.True(t, created[1].stopped)
	require.False(t, controller.Running())
	buffer.Reset()

	require.NoError(t, runDaemonAction("stop", controller, buffer))
	require.Contains(t, buffer.String(), "daemon is not running")
}

func TestRunDaemonActionServeStartsDaemon(t *testing.T) {
	controller := newDaemonController(func() managedDaemon {
		return &managedDaemonStub{}
	})

	buffer := new(bytes.Buffer)
	require.NoError(t, runDaemonAction("serve", controller, buffer))
	require.Contains(t, buffer.String(), "daemon started")
	require.True(t, controller.Running())
}

func TestRunDaemonActionUnknown(t *testing.T) {
	controller := newDaemonController(func() managedDaemon {
		return &managedDaemonStub{}
	})

	err := runDaemonAction("invalid", controller, new(bytes.Buffer))
	require.Error(t, err)
	require.ErrorContains(t, err, "unknown daemon action")
}
