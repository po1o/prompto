package daemon

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	libruntime "runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jandedobbeleer/oh-my-posh/src/runtime"

	"github.com/stretchr/testify/require"
)

func TestDaemonStartRenderAndNextUpdateFlow(t *testing.T) {
	daemon := New(&rendererStub{})
	sessionID := strconv.Itoa(os.Getpid())

	initial := daemon.StartRender(RenderRequest{
		SessionID: sessionID,
		Flags:     &runtime.Flags{},
	})
	require.Equal(t, "initial", initial.Type)
	require.Equal(t, "render", initial.Bundle.Primary)

	go func() {
		time.Sleep(20 * time.Millisecond)
		daemon.SessionHub(sessionID).Publish("path.main")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	update, ok := daemon.NextUpdate(ctx, sessionID, 0)
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
			SessionID: strconv.Itoa(os.Getpid()),
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

func TestDaemonReloadWaitsForActiveRenderCompletion(t *testing.T) {
	daemon := New(&rendererStub{})
	sessionID := strconv.Itoa(os.Getpid())

	daemon.StartRender(RenderRequest{
		SessionID: sessionID,
		Flags:     &runtime.Flags{},
	})

	reloadDone := make(chan struct{})
	go func() {
		daemon.Reload(nil)
		close(reloadDone)
	}()

	select {
	case <-reloadDone:
		t.Fatal("reload should wait for active render completion")
	case <-time.After(50 * time.Millisecond):
	}

	daemon.CompleteSession(sessionID)

	select {
	case <-reloadDone:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("reload should complete after active render completion")
	}
}

func TestDaemonStopPreventsNewOperations(t *testing.T) {
	daemon := New(&rendererStub{})
	daemon.Stop()

	initial := daemon.StartRender(RenderRequest{
		SessionID: strconv.Itoa(os.Getpid()),
		Flags:     &runtime.Flags{},
	})
	require.Equal(t, "stopped", initial.Type)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, ok := daemon.NextUpdate(ctx, "session-a", 0)
	require.False(t, ok)
}

func TestDaemonAutoStopsAfterIdleTimeoutWithoutTrackedSessions(t *testing.T) {
	daemon := NewWithIdleTimeout(25*time.Millisecond, &rendererStub{})

	time.Sleep(60 * time.Millisecond)

	response := daemon.StartRender(RenderRequest{
		SessionID: "101",
		Flags:     &runtime.Flags{},
	})
	require.Equal(t, "stopped", response.Type)
}

func TestDaemonNewFromConfigUsesConfiguredIdleTimeout(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "daemon.omp.json")
	err := os.WriteFile(configPath, []byte(`{"version":4,"daemon_idle_timeout":"2"}`), 0o644)
	require.NoError(t, err)

	daemon := NewFromConfig(configPath, &rendererStub{})
	require.Equal(t, 2*time.Minute, daemon.idleTimeout)
}

func TestDaemonIdleTimerStartsAfterTrackedSessionCompletion(t *testing.T) {
	daemon := NewWithIdleTimeout(30*time.Millisecond, &rendererStub{})
	sessionID := strconv.Itoa(os.Getpid())

	initial := daemon.StartRender(RenderRequest{
		SessionID: sessionID,
		Flags:     &runtime.Flags{},
	})
	require.Equal(t, "initial", initial.Type)

	time.Sleep(50 * time.Millisecond)
	daemon.CompleteSession(sessionID)
	time.Sleep(70 * time.Millisecond)

	stopped := daemon.StartRender(RenderRequest{
		SessionID: "103",
		Flags:     &runtime.Flags{},
	})
	require.Equal(t, "stopped", stopped.Type)
}

func TestDaemonDoesNotStopWhileTrackedPIDIsAlive(t *testing.T) {
	daemon := NewWithIdleTimeout(25*time.Millisecond, &rendererStub{})
	sessionID := strconv.Itoa(os.Getpid())

	daemon.StartRender(RenderRequest{
		SessionID: sessionID,
		Flags:     &runtime.Flags{},
	})

	require.Eventually(t, func() bool {
		response := daemon.StartRender(RenderRequest{
			SessionID: sessionID,
			Flags:     &runtime.Flags{},
		})
		return response.Type == "initial"
	}, 150*time.Millisecond, 10*time.Millisecond)
}

func TestDaemonStopsAfterProcessExitForTrackedPID(t *testing.T) {
	daemon := NewWithIdleTimeout(30*time.Millisecond, &rendererStub{})
	sessionID := "99999999"

	daemon.StartRender(RenderRequest{
		SessionID: sessionID,
		Flags:     &runtime.Flags{},
	})

	time.Sleep(70 * time.Millisecond)

	response := daemon.StartRender(RenderRequest{
		SessionID: "101",
		Flags:     &runtime.Flags{},
	})
	require.Equal(t, "stopped", response.Type)
}

func TestDaemonStopsAfterTrackedProcessActuallyExits(t *testing.T) {
	pid := startDetachedTestProcessPID(t)
	daemon := NewWithIdleTimeout(30*time.Millisecond, &rendererStub{})

	initial := daemon.StartRender(RenderRequest{
		SessionID: strconv.Itoa(pid),
		Flags:     &runtime.Flags{},
	})
	require.Equal(t, "initial", initial.Type)

	require.Eventually(t, func() bool {
		return daemon.stopped.Load()
	}, 3*time.Second, 20*time.Millisecond)
}

func TestCompleteSessionForNonNumericIDDoesNotAffectTrackedPID(t *testing.T) {
	daemon := NewWithIdleTimeout(200*time.Millisecond, &rendererStub{})
	trackedSessionID := strconv.Itoa(os.Getpid())

	daemon.StartRender(RenderRequest{
		SessionID: trackedSessionID,
		Flags:     &runtime.Flags{},
	})
	daemon.StartRender(RenderRequest{
		SessionID: "nonnumeric",
		Flags:     &runtime.Flags{},
	})

	daemon.CompleteSession("nonnumeric")

	require.Eventually(t, func() bool {
		response := daemon.StartRender(RenderRequest{
			SessionID: trackedSessionID,
			Flags:     &runtime.Flags{},
		})
		return response.Type == "initial"
	}, 300*time.Millisecond, 20*time.Millisecond)
}

func TestParseSessionPID(t *testing.T) {
	pid, ok := parseSessionPID("1234")
	require.True(t, ok)
	require.Equal(t, 1234, pid)

	_, ok = parseSessionPID("0")
	require.False(t, ok)

	_, ok = parseSessionPID("-1")
	require.False(t, ok)

	_, ok = parseSessionPID("not-a-pid")
	require.False(t, ok)
}

func startDetachedTestProcessPID(t *testing.T) int {
	t.Helper()

	command, args := detachedProcessPIDCommand()
	cmd := exec.CommandContext(context.Background(), command, args...)
	output, err := cmd.Output()
	require.NoError(t, err)

	pidText := strings.TrimSpace(string(output))
	pid, err := strconv.Atoi(pidText)
	require.NoError(t, err)
	require.Greater(t, pid, 0)

	return pid
}

func detachedProcessPIDCommand() (string, []string) {
	if libruntime.GOOS == runtime.WINDOWS {
		return "powershell", []string{
			"-NoProfile",
			"-Command",
			"$p = Start-Process -FilePath powershell -ArgumentList '-NoProfile -Command Start-Sleep -Seconds 1' -PassThru; $p.Id",
		}
	}

	return "sh", []string{"-c", "sleep 1 & echo $!"}
}
