package daemon

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionManager(t *testing.T) {
	called := false
	sm := NewSessionManager(func(_ int) { called = true }, nil)

	assert.NotNil(t, sm)
	assert.Equal(t, 0, sm.Count())
	assert.False(t, called)
}

func TestSessionManagerRegister(t *testing.T) {
	sm := NewSessionManager(nil, nil)

	pid := os.Getpid()
	sm.Register(pid, "uuid", "shell")
	assert.Equal(t, 1, sm.Count())

	sm.Register(pid, "uuid", "shell")
	assert.Equal(t, 1, sm.Count())
}

func TestSessionManagerUnregister(t *testing.T) {
	var unregisterCount atomic.Int32
	sm := NewSessionManager(func(_ int) { unregisterCount.Add(1) }, nil)

	pid := os.Getpid()
	sm.Register(pid, "uuid", "shell")
	sm.Unregister(pid)

	assert.Equal(t, 0, sm.Count())
	assert.Equal(t, int32(1), unregisterCount.Load())
}

func TestSessionManagerOnEmptyCallback(t *testing.T) {
	var emptyCount atomic.Int32
	sm := NewSessionManager(nil, func() { emptyCount.Add(1) })

	pid := os.Getpid()
	sm.Register(pid, "uuid", "shell")
	sm.Unregister(pid)

	assert.Equal(t, int32(1), emptyCount.Load())
}

func TestSessionManagerDetectsExitedProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping process exit detection test in short mode")
	}

	var emptyCount atomic.Int32
	sm := NewSessionManager(nil, func() { emptyCount.Add(1) })

	process := startSessionTestProcess(t)
	sm.Register(process.Pid, "uuid", "shell")

	assert.Eventually(t, func() bool {
		return sm.Count() == 0 && emptyCount.Load() == 1
	}, 2*time.Second, 20*time.Millisecond)
}

func TestSessionManagerNonExistentPID(t *testing.T) {
	var emptyCount atomic.Int32
	sm := NewSessionManager(nil, func() { emptyCount.Add(1) })

	sm.Register(999999999, "uuid", "shell")

	assert.Eventually(t, func() bool {
		return sm.Count() == 0 && emptyCount.Load() == 1
	}, time.Second, 20*time.Millisecond)
}

func startSessionTestProcess(t *testing.T) *os.Process {
	t.Helper()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(context.Background(), "ping", "-n", "1", "127.0.0.1")
	}
	if runtime.GOOS != "windows" {
		cmd = exec.CommandContext(context.Background(), "sleep", "0.1")
	}

	err := cmd.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	return cmd.Process
}
