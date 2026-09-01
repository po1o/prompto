package daemon

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/po1o/prompto/src/daemon/ipc"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// The client's connection and RPC wrappers need an in-process gRPC harness
// to exercise (the existing server_test.go style). What is cheap to test
// here is ExtractPrompts — pure response parsing with zero side effects and
// a long history of B1 flagging it at 0% coverage.

func TestExtractPromptsNilResponse(t *testing.T) {
	result := ExtractPrompts(nil)
	require.NotNil(t, result)
	require.Equal(t, PromptResult{}, *result)
}

func TestExtractPromptsEmptyPrompts(t *testing.T) {
	result := ExtractPrompts(&ipc.PromptResponse{})
	require.NotNil(t, result)
	require.Equal(t, PromptResult{}, *result)
}

func TestExtractPromptsAllFieldsPopulated(t *testing.T) {
	result := ExtractPrompts(&ipc.PromptResponse{
		Prompts: map[string]*ipc.Prompt{
			"primary":    {Text: "P"},
			"right":      {Text: "R"},
			"secondary":  {Text: "S"},
			"transient":  {Text: "T"},
			"rtransient": {Text: "RT"},
			"debug":      {Text: "D"},
			"tooltip":    {Text: "TT"},
			"valid":      {Text: "V"},
			"error":      {Text: "E"},
		},
	})

	require.Equal(t, &PromptResult{
		Primary:    "P",
		Right:      "R",
		Secondary:  "S",
		Transient:  "T",
		RTransient: "RT",
		Debug:      "D",
		Tooltip:    "TT",
		Valid:      "V",
		Error:      "E",
	}, result)
}

func TestExtractPromptsPartialFieldsPopulated(t *testing.T) {
	result := ExtractPrompts(&ipc.PromptResponse{
		Prompts: map[string]*ipc.Prompt{
			"primary":   {Text: "P"},
			"transient": {Text: "T"},
		},
	})

	require.Equal(t, &PromptResult{
		Primary:   "P",
		Transient: "T",
	}, result)
}

func TestExtractPromptsIgnoresUnknownKeys(t *testing.T) {
	result := ExtractPrompts(&ipc.PromptResponse{
		Prompts: map[string]*ipc.Prompt{
			"primary":     {Text: "P"},
			"unknown_key": {Text: "ignored"},
		},
	})

	require.Equal(t, &PromptResult{Primary: "P"}, result)
}

// TestConnectOrStartWaitsForTheDaemonToAcceptConnections covers the cold start.
// The daemon loads its config before it listens, so its socket is not bound the
// moment the process exists — measured starts take a few hundred milliseconds.
// A connect to an unbound socket fails at once rather than blocking, so waiting
// a fixed 50ms and trying once gave up while the daemon was still coming up,
// and the first prompt after a daemon restart failed outright.
func TestConnectOrStartWaitsForTheDaemonToAcceptConnections(t *testing.T) {
	socketDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", socketDir)
	t.Setenv("XDG_STATE_HOME", socketDir)

	// Longer than any fixed sleep short enough to keep a prompt responsive.
	const startupDelay = 250 * time.Millisecond

	server := grpc.NewServer()
	t.Cleanup(server.Stop)

	startFunc := func() error {
		go func() {
			time.Sleep(startupDelay)

			listener, err := ipc.Listen()
			if err != nil {
				return
			}

			_ = server.Serve(listener)
		}()

		return nil
	}

	start := time.Now()
	client, err := ConnectOrStart(startFunc)
	require.NoError(t, err, "gave up while the daemon was still starting")
	require.NotNil(t, client)
	t.Cleanup(func() { _ = client.Close() })

	require.GreaterOrEqual(t, time.Since(start), startupDelay,
		"cannot have connected before the daemon was listening")
}

// TestConnectOrStartReportsAFailureToSpawn keeps a daemon that cannot be
// started from being retried against; there is nothing coming up to wait for.
func TestConnectOrStartReportsAFailureToSpawn(t *testing.T) {
	socketDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", socketDir)
	t.Setenv("XDG_STATE_HOME", socketDir)

	start := time.Now()
	client, err := ConnectOrStart(func() error { return errors.New("no binary") })

	require.Error(t, err)
	require.Nil(t, client)
	require.Less(t, time.Since(start), startTimeout, "must not wait for a daemon that was never started")
}

// startFakeDaemonProcess runs a long-lived process under a name
// IsProcessRunning accepts, so a lock file can name a genuinely live daemon
// without standing up a real one. Using a stand-in rather than the test's own
// PID means a regression shows up as a failed assertion instead of the test
// binary killing itself.
func startFakeDaemonProcess(t *testing.T) *exec.Cmd {
	t.Helper()

	sleep, err := exec.LookPath("sleep")
	require.NoError(t, err)

	binary, err := os.ReadFile(sleep)
	require.NoError(t, err)

	// IsProcessRunning reads the executable name to guard against PID reuse.
	fake := filepath.Join(t.TempDir(), "prompto-fake-daemon")
	require.NoError(t, os.WriteFile(fake, binary, 0o700))

	cmd := exec.CommandContext(t.Context(), fake, "60")
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	require.True(t, IsProcessRunning(cmd.Process.Pid), "fake daemon must look alive to the lock code")

	return cmd
}

func writeDaemonLock(t *testing.T, pid int) string {
	t.Helper()

	path := lockFilePath()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, fmt.Appendf(nil, "%d\n", pid), 0o600))

	return path
}

// serveAfter binds the daemon socket once delay has passed, standing in for a
// daemon that is up but not yet answering.
func serveAfter(t *testing.T, delay time.Duration) {
	t.Helper()

	server := grpc.NewServer()
	t.Cleanup(server.Stop)

	go func() {
		time.Sleep(delay)

		listener, err := ipc.Listen()
		if err != nil {
			return
		}

		_ = server.Serve(listener)
	}()
}

// TestConnectOrStartDoesNotReplaceAReachableDaemon covers the shared daemon. A
// first connect can fail against a perfectly healthy one — it may be busy, or
// another shell may have spawned it moments ago and it is still binding its
// socket. Killing it there takes it out from under every other shell using it
// and makes their next prompt pay the restart, so only a daemon that stays
// unreachable is replaced.
func TestConnectOrStartDoesNotReplaceAReachableDaemon(t *testing.T) {
	socketDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", socketDir)
	t.Setenv("XDG_STATE_HOME", socketDir)

	daemonProcess := startFakeDaemonProcess(t)
	lockPath := writeDaemonLock(t, daemonProcess.Process.Pid)

	// It answers a moment after the first attempt has already failed.
	serveAfter(t, 150*time.Millisecond)

	var spawned atomic.Bool
	client, err := ConnectOrStart(func() error {
		spawned.Store(true)
		return nil
	})

	require.NoError(t, err)
	require.NotNil(t, client)
	t.Cleanup(func() { _ = client.Close() })

	require.False(t, spawned.Load(), "started a second daemon over one that was reachable")
	require.True(t, IsProcessRunning(daemonProcess.Process.Pid), "killed a daemon that was reachable")
	require.FileExists(t, lockPath, "removed the lock of a daemon that was reachable")
}

// TestConnectOrStartReplacesADaemonThatGoesAway covers a daemon shutting down
// as we reach for it — an upgraded binary, or an idle timeout. There is nothing
// left to wait for, so the wait ends as soon as the process is gone and a fresh
// daemon is started rather than the deadline being sat out.
func TestConnectOrStartReplacesADaemonThatGoesAway(t *testing.T) {
	socketDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", socketDir)
	t.Setenv("XDG_STATE_HOME", socketDir)

	daemonProcess := startFakeDaemonProcess(t)
	writeDaemonLock(t, daemonProcess.Process.Pid)

	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = daemonProcess.Process.Kill()
		// Reap it: a zombie still answers kill(pid, 0).
		_, _ = daemonProcess.Process.Wait()
	}()

	var spawned atomic.Bool
	start := time.Now()
	client, err := ConnectOrStart(func() error {
		spawned.Store(true)
		serveAfter(t, 0)
		return nil
	})

	require.NoError(t, err)
	require.NotNil(t, client)
	t.Cleanup(func() { _ = client.Close() })

	require.True(t, spawned.Load(), "never replaced a daemon that had gone")
	require.Less(t, time.Since(start), startTimeout, "sat out the deadline for a daemon that had gone")
}
