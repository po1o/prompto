package daemon

import (
	"context"
	"os"
	"path/filepath"
	libruntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/daemon/ipc"
	"github.com/stretchr/testify/require"
)

const windowsOS = "windows"

func TestServerToggleSegmentIsSessionScoped(t *testing.T) {
	socketDir := testSocketDir(t)
	t.Setenv("XDG_STATE_HOME", socketDir)
	t.Setenv("XDG_RUNTIME_DIR", socketDir)

	configPath := filepath.Join(t.TempDir(), "daemon-toggle.omp.yaml")
	configYAML := `
blocks:
  - type: prompt
    segments:
      - type: text
        alias: left
        template: A
      - type: text
        alias: right
        template: B
`

	require.NoError(t, os.WriteFile(configPath, []byte(configYAML), 0o644))

	server := startTestServer(t, configPath)
	client := newDaemonServiceClient(t)

	_, err := client.ToggleSegment(context.Background(), &ipc.ToggleSegmentRequest{
		SessionId: "session-one",
		Segments:  []string{"left"},
	})
	require.NoError(t, err)

	sessionOneToggles := server.sessionToggles("session-one")
	require.True(t, sessionOneToggles["left"])

	sessionTwoToggles := server.sessionToggles("session-two")
	require.False(t, sessionTwoToggles["left"])

	_, err = client.ToggleSegment(context.Background(), &ipc.ToggleSegmentRequest{
		SessionId: "session-one",
		Segments:  []string{"left"},
	})
	require.NoError(t, err)

	sessionOneToggles = server.sessionToggles("session-one")
	require.False(t, sessionOneToggles["left"])

	stopTestServer(t, server)
}

func TestServerSetLoggingWritesFile(t *testing.T) {
	socketDir := testSocketDir(t)
	t.Setenv("XDG_STATE_HOME", socketDir)
	t.Setenv("XDG_RUNTIME_DIR", socketDir)

	configPath := filepath.Join(t.TempDir(), "daemon-log.omp.yaml")
	configYAML := `
blocks:
  - type: prompt
    segments:
      - type: text
        template: LOG
`

	require.NoError(t, os.WriteFile(configPath, []byte(configYAML), 0o644))

	logPath := filepath.Join(t.TempDir(), "daemon.log")
	server := startTestServer(t, configPath)
	client := newDaemonServiceClient(t)

	response, err := client.SetLogging(context.Background(), &ipc.SetLoggingRequest{Path: logPath})
	require.NoError(t, err)
	require.True(t, response.Success)

	require.Eventually(t, func() bool {
		data, readErr := os.ReadFile(logPath)
		if readErr != nil {
			return false
		}

		return len(data) > 0 && strings.Contains(string(data), "DEBUG")
	}, 2*time.Second, 50*time.Millisecond)

	response, err = client.SetLogging(context.Background(), &ipc.SetLoggingRequest{})
	require.NoError(t, err)
	require.True(t, response.Success)

	stopTestServer(t, server)
}

func TestResolveServerConfigPathUsesXDGConfigHomeByDefault(t *testing.T) {
	if libruntime.GOOS == windowsOS {
		t.Skip("XDG config home is not used on windows")
	}

	xdgConfigHome := filepath.Join(t.TempDir(), "xdg-config")
	t.Setenv("XDG_CONFIG_HOME", xdgConfigHome)
	t.Setenv("HOME", "")

	resolved := resolveServerConfigPath("")
	expected := filepath.Join(xdgConfigHome, "prompto", "config.yaml")
	require.Equal(t, filepath.Clean(expected), filepath.Clean(resolved))
}

func TestResolveServerConfigPathFallsBackToHomeDotConfig(t *testing.T) {
	if libruntime.GOOS == windowsOS {
		t.Skip("home fallback path differs on windows")
	}

	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)

	resolved := resolveServerConfigPath("")
	expected := filepath.Join(home, ".config", "prompto", "config.yaml")
	require.Equal(t, filepath.Clean(expected), filepath.Clean(resolved))
}

func TestProcessPendingConfigReloadAppliesQueuedReloadBeforeRender(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "daemon-reload.omp.yaml")
	configBody := `
prompt:
  - segments: ["text.main"]

text.main:
  type: text
  template: A
`
	require.NoError(t, os.WriteFile(configPath, []byte(configBody), 0o644))

	deviceCache := NewDeviceCache()
	server := &Server{
		configPath:     configPath,
		core:           NewFromConfigWithDeviceCache(configPath, nil, deviceCache),
		deviceCache:    deviceCache,
		configReloadCh: make(chan struct{}, 1),
	}
	t.Cleanup(func() {
		server.core.Stop()
		cache.Delete(cache.Device, config.RELOAD)
	})

	cache.Delete(cache.Device, config.RELOAD)
	server.requestConfigReload(configPath)
	server.processPendingConfigReload()

	reloadFlag, ok := cache.Get[bool](cache.Device, config.RELOAD)
	require.True(t, ok)
	require.True(t, reloadFlag)
	require.Equal(t, 0, len(server.configReloadCh))
}

func TestReloadIfConfigFileUpdatedAppliesReloadWithoutQueuedEvent(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "daemon-reload-mtime.omp.yaml")
	configBody := `
prompt:
  - segments: ["text.main"]

text.main:
  type: text
  template: A
`
	require.NoError(t, os.WriteFile(configPath, []byte(configBody), 0o644))

	deviceCache := NewDeviceCache()
	server := &Server{
		configPath:     configPath,
		core:           NewFromConfigWithDeviceCache(configPath, nil, deviceCache),
		deviceCache:    deviceCache,
		configReloadCh: make(chan struct{}, 1),
	}
	t.Cleanup(func() {
		server.core.Stop()
		cache.Delete(cache.Device, config.RELOAD)
	})

	server.captureConfigModTime()
	cache.Delete(cache.Device, config.RELOAD)

	time.Sleep(15 * time.Millisecond)
	configBody = `
prompt:
  - segments: ["text.main"]

text.main:
  type: text
  template: B
`
	require.NoError(t, os.WriteFile(configPath, []byte(configBody), 0o644))

	server.reloadIfConfigFileUpdated()

	reloadFlag, ok := cache.Get[bool](cache.Device, config.RELOAD)
	require.True(t, ok)
	require.True(t, reloadFlag)
	require.Equal(t, 0, len(server.configReloadCh))
}

func TestMakePromptResponseIncludesRightTransientWhenPresent(t *testing.T) {
	response := makePromptResponse("update", "request-1", &PromptBundle{
		Primary:    "left",
		RPrompt:    "right",
		Transient:  "transient-left",
		RTransient: "transient-right",
	})

	require.NotNil(t, response)
	require.Equal(t, "transient-right", response.Prompts["rtransient"].Text)
}

func TestServerReplacePrimaryStreamCancelsPrevious(t *testing.T) {
	server := &Server{
		primaryStreams: make(map[string]primaryStreamState),
	}

	firstCanceled := make(chan struct{}, 1)
	firstRelease := server.replacePrimaryStream("session-a", "request-1", func() {
		select {
		case firstCanceled <- struct{}{}:
		default:
		}
	})

	secondRelease := server.replacePrimaryStream("session-a", "request-2", func() {})

	select {
	case <-firstCanceled:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected previous primary stream to be canceled")
	}

	firstRelease()

	server.streamMu.Lock()
	current, ok := server.primaryStreams["session-a"]
	server.streamMu.Unlock()
	require.True(t, ok)
	require.Equal(t, "request-2", current.requestID)

	secondRelease()

	server.streamMu.Lock()
	_, ok = server.primaryStreams["session-a"]
	server.streamMu.Unlock()
	require.False(t, ok)
}

func startTestServer(t *testing.T, configPath string) *Server {
	t.Helper()

	server, err := NewServer(configPath)
	require.NoError(t, err)

	errChannel := make(chan error, 1)
	go func() {
		errChannel <- server.Start()
	}()

	require.Eventually(t, ipc.SocketExists, 2*time.Second, 50*time.Millisecond)

	select {
	case startErr := <-errChannel:
		require.NoError(t, startErr)
	default:
	}

	return server
}

func stopTestServer(t *testing.T, server *Server) {
	t.Helper()

	server.Stop()

	select {
	case <-server.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop in time")
	}
}

func newDaemonServiceClient(t *testing.T) ipc.DaemonServiceClient {
	t.Helper()

	conn, err := ipc.Dial()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = conn.Close()
	})

	return ipc.NewDaemonServiceClient(conn)
}

func testSocketDir(t *testing.T) string {
	t.Helper()

	if libruntime.GOOS == windowsOS {
		return t.TempDir()
	}

	directory, err := os.MkdirTemp("/tmp", "omp")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(directory)
	})

	return directory
}
