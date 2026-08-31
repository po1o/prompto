package daemon

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/prompt"
	"github.com/po1o/prompto/src/runtime"

	"github.com/stretchr/testify/require"
)

const sessionIDFixture = "session-a"

func newActiveRenderTestDaemon() *Daemon {
	// A real Daemon configured with a stub engine factory and stub renderer,
	// so render tests don't spin up the full prompt engine. Keeps the idle
	// timer so existing idle-stop tests still exercise it.
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	pipeline := NewRenderPipeline(registry, NewReloadGate(), &rendererStub{}, nil)
	daemon := &Daemon{
		pipeline:    pipeline,
		deviceCache: NewDeviceCache(),
		renders:     make(map[string]*ActiveRender),
		idleTimeout: 5 * time.Minute,
	}
	daemon.sessions = NewProcessTracker(daemon.onSessionUnregister, daemon.onAllSessionsEnded)
	return daemon
}

func TestDaemonStartRenderCompletesSynchronouslyWhenPrimaryHasNoPendingUpdates(t *testing.T) {
	daemon := New(&rendererStub{})
	sessionID := strconv.Itoa(os.Getpid())

	initial := daemon.StartRender(RenderRequest{
		SessionID: sessionID,
		Flags:     &runtime.Flags{},
	})
	require.Equal(t, "initial", initial.Type)
	require.Equal(t, "render", initial.Bundle.Primary)
	require.Equal(t, "transient", initial.Bundle.Transient)
	require.Equal(t, "rtransient", initial.Bundle.RTransient)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, ok := daemon.NextUpdate(ctx, sessionID, 0)
	require.False(t, ok)
}

func TestDaemonStartRenderAndNextUpdateFlow(t *testing.T) {
	daemon := newActiveRenderTestDaemon()
	sessionID := sessionIDFixture

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
	daemon := newActiveRenderTestDaemon()
	sessionID := sessionIDFixture

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
	_, ok := daemon.NextUpdate(ctx, sessionIDFixture, 0)
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
	configPath := filepath.Join(t.TempDir(), "daemon.omp.yaml")
	err := os.WriteFile(configPath, []byte("daemon_idle_timeout: \"2\"\n"), 0o644)
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
	}, 5*time.Second, 20*time.Millisecond)
}

func TestDaemonIdleStopInvokesStopCallback(t *testing.T) {
	daemon := NewWithIdleTimeout(25*time.Millisecond, &rendererStub{})
	stopped := make(chan struct{}, 1)
	daemon.SetOnStop(func() {
		select {
		case stopped <- struct{}{}:
		default:
		}
	})

	select {
	case <-stopped:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("daemon stop callback was not invoked")
	}
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
	err := cmd.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		if cmd.Process == nil {
			return
		}

		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	require.NotNil(t, cmd.Process)
	require.Greater(t, cmd.Process.Pid, 0)

	return cmd.Process.Pid
}

func detachedProcessPIDCommand() (string, []string) {
	return "sleep", []string{"1"}
}

// Segment.isToggled treats a session's toggle map as authoritative whenever it
// is non-nil, so SessionToggles must never hand back nil: a nil map sends the
// render back to the shared toggle cache, where the config's `toggled: true`
// seeds live, and makes such a segment impossible to switch on again.
func TestSessionTogglesNeverReturnsNil(t *testing.T) {
	clearSharedToggleCache(t)

	daemon := New(&rendererStub{})

	require.NotNil(t, daemon.SessionToggles("never-seen"))

	daemon.ToggleSegment(sessionIDFixture, []string{"git"})
	require.True(t, daemon.SessionToggles(sessionIDFixture)["git"])

	// Toggling the only entry back on leaves an empty map, never a nil one.
	daemon.ToggleSegment(sessionIDFixture, []string{"git"})
	toggles := daemon.SessionToggles(sessionIDFixture)
	require.NotNil(t, toggles)
	require.False(t, toggles["git"])
}

// Pids are recycled, so a session's toggles must not outlive its shell. The
// next shell landing on that pid would inherit them, and since even an empty
// map is authoritative it would render a `toggled: true` segment it never
// toggled — or keep one hidden that it never turned off.
func TestCompleteSessionForgetsSessionToggles(t *testing.T) {
	clearSharedToggleCache(t)

	daemon := New(&rendererStub{})
	sessionID := strconv.Itoa(os.Getpid())

	daemon.ToggleSegment(sessionID, []string{"git"})
	require.True(t, daemon.SessionToggles(sessionID)["git"])

	daemon.CompleteSession(sessionID)

	daemon.toggleMu.RLock()
	_, stored := daemon.segmentToggles[sessionID]
	daemon.toggleMu.RUnlock()
	require.False(t, stored, "toggles must not outlive the session")

	// A shell reusing the pid is seeded afresh from the shared cache.
	require.False(t, daemon.SessionToggles(sessionID)["git"])
}

// A shell that exits for real must lose its toggles, not just one taken down
// through CompleteSession: onSessionUnregister is the only route into
// completeRender that production actually uses.
func TestExitedProcessForgetsSessionToggles(t *testing.T) {
	clearSharedToggleCache(t)

	daemon := New(&rendererStub{})
	pid := startDetachedTestProcessPID(t)
	sessionID := strconv.Itoa(pid)

	daemon.StartRender(RenderRequest{
		SessionID: sessionID,
		Flags:     &runtime.Flags{},
	})

	daemon.ToggleSegment(sessionID, []string{"git"})
	require.True(t, daemon.SessionToggles(sessionID)["git"])

	require.NoError(t, exec.Command("kill", strconv.Itoa(pid)).Run())

	require.Eventually(t, func() bool {
		daemon.toggleMu.RLock()
		defer daemon.toggleMu.RUnlock()
		_, stored := daemon.segmentToggles[sessionID]
		return !stored
	}, 5*time.Second, 10*time.Millisecond, "toggles must not outlive the shell")
}

// The shared toggle cache is process-global and any test that loads a config
// writes it, so tests asserting on seeded toggles must start from a known
// state rather than inherit whatever ran before them.
func clearSharedToggleCache(t *testing.T) {
	t.Helper()

	cache.Delete(cache.Session, cache.TOGGLECACHE)
	t.Cleanup(func() {
		cache.Delete(cache.Session, cache.TOGGLECACHE)
	})
}

// A `toggled: true` the config gains must reach shells that are already
// running — before the seed delta they kept rendering the segment until they
// exited. The converse must not happen: a toggle the user cleared stays
// cleared across the reload rather than being seeded back on.
func TestConfigReloadPushesOnlyNewlyAddedToggles(t *testing.T) {
	clearSharedToggleCache(t)

	configPath := filepath.Join(t.TempDir(), "reload-toggles.prompto.yaml")
	write := func(body string) {
		t.Helper()
		require.NoError(t, os.WriteFile(configPath, []byte(body), 0o644))
	}

	write(`
left:
    type: text
    template: A
    toggled: true
right:
    type: text
    template: B
prompt:
    - segments:
        - left
        - right
`)

	daemon := newConfigTestDaemon(t, configPath)

	// The session inherits the config's toggle, then the user switches it on.
	require.True(t, daemon.SessionToggles(sessionIDFixture)["left"])
	daemon.ToggleSegment(sessionIDFixture, []string{"left"})
	require.False(t, daemon.SessionToggles(sessionIDFixture)["left"])

	write(`
left:
    type: text
    template: A
    toggled: true
right:
    type: text
    template: B
    toggled: true
prompt:
    - segments:
        - left
        - right
`)

	daemon.applyConfigReload()

	toggles := daemon.SessionToggles(sessionIDFixture)
	require.True(t, toggles["right"], "a newly configured toggle must reach a live session")
	require.False(t, toggles["left"], "a toggle the user cleared must not be seeded back on")
}

// Clearing the cache empties the shared toggle cache the seed snapshot
// describes. The snapshot has to go with it, or the config's `toggled: true`
// looks already-seeded forever after and never reaches a session again.
func TestResetTogglesLetsTheConfigReseedLiveSessions(t *testing.T) {
	clearSharedToggleCache(t)

	configPath := filepath.Join(t.TempDir(), "reset-toggles.prompto.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(`
left:
    type: text
    template: A
    toggled: true
prompt:
    - segments:
        - left
`), 0o644))

	daemon := newConfigTestDaemon(t, configPath)

	require.True(t, daemon.SessionToggles(sessionIDFixture)["left"])

	// What the CacheClear RPC does: drop the toggles and the shared cache.
	daemon.ResetToggles()
	cache.DeleteAll(cache.Session)

	// A render lands before the config is reloaded, so the session re-seeds
	// from the now-empty cache and holds an (authoritative) empty map.
	require.False(t, daemon.SessionToggles(sessionIDFixture)["left"])

	daemon.applyConfigReload()

	require.True(t, daemon.SessionToggles(sessionIDFixture)["left"], "the config must be able to seed toggles again after a cache clear")
}

// newConfigTestDaemon binds a daemon to a config path without starting the
// fsnotify watcher. These tests drive applyConfigReload themselves, and a live
// reload worker would fire on their own file writes — running a second,
// concurrent reload whose config.Load races the test's.
func newConfigTestDaemon(t *testing.T, configPath string) *Daemon {
	t.Helper()

	// Mirrors NewFromConfigWithDeviceCache: the config must be loaded before
	// the constructor, which snapshots the toggles it seeded.
	cfg := config.Load(configPath)
	daemon := NewWithIdleTimeoutAndDeviceCache(cfg.GetDaemonIdleTimeout(), &rendererStub{}, nil)
	daemon.configPath = configPath
	t.Cleanup(daemon.Stop)

	return daemon
}

// A shell's first render and a `prompto toggle` for that same shell can
// overlap: both seed the session. Whichever seeds second must not overwrite
// the first, or the toggle command silently does nothing.
func TestSessionTogglesDoesNotDropAConcurrentToggle(t *testing.T) {
	clearSharedToggleCache(t)

	daemon := New(&rendererStub{})

	for i := range 2000 {
		sessionID := strconv.Itoa(i)

		var wg sync.WaitGroup
		wg.Go(func() {
			daemon.SessionToggles(sessionID)
		})
		wg.Go(func() {
			daemon.ToggleSegment(sessionID, []string{"git"})
		})
		wg.Wait()

		require.True(t, daemon.SessionToggles(sessionID)["git"], "toggle lost to a concurrent first render (session %s)", sessionID)
	}
}
