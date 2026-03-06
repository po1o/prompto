package daemon

import (
	"context"
	"os"
	"path/filepath"
	libruntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/jandedobbeleer/oh-my-posh/src/daemon/ipc"
	"github.com/stretchr/testify/require"
)

const windowsOS = "windows"

func TestServerToggleSegmentIsSessionScoped(t *testing.T) {
	socketDir := testSocketDir(t)
	t.Setenv("XDG_STATE_HOME", socketDir)
	t.Setenv("XDG_RUNTIME_DIR", socketDir)

	configPath := filepath.Join(t.TempDir(), "daemon-toggle.omp.yaml")
	config := `
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

	require.NoError(t, os.WriteFile(configPath, []byte(config), 0o644))

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
	config := `
blocks:
  - type: prompt
    segments:
      - type: text
        template: LOG
`

	require.NoError(t, os.WriteFile(configPath, []byte(config), 0o644))

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
