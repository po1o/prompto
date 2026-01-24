package daemon

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jandedobbeleer/oh-my-posh/src/daemon/ipc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createVimTestConfig(t *testing.T) string {
	t.Helper()
	content := `{
		"version": 4,
		"blocks": [
			{
				"type": "prompt",
				"alignment": "left",
				"segments": [
					{
						"type": "vim",
						"style": "plain",
						"template": "{{ .Mode }}"
					}
				]
			}
		]
	}`
	tmpFile, err := os.CreateTemp("", "omp-vim-config-*.json")
	require.NoError(t, err)
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	err = tmpFile.Close()
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })
	return tmpFile.Name()
}

func TestVimModeUpdateOnRepaint(t *testing.T) {
	tmpDir := testSocketDir(t)
	t.Setenv("XDG_STATE_HOME", tmpDir)
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	d, err := New(createVimTestConfig(t))
	require.NoError(t, err)

	go func() { _ = d.Start() }()
	time.Sleep(100 * time.Millisecond)

	defer func() {
		d.shutdown()
		<-d.Done()
	}()

	conn, err := ipc.Dial()
	require.NoError(t, err)
	defer conn.Close()

	client := ipc.NewDaemonServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Initial Render: INSERT mode
	stream1, err := client.RenderPrompt(ctx, &ipc.PromptRequest{
		Version:   ipc.ProtocolVersion,
		SessionId: "vim-test-session",
		RequestId: "req-1",
		Flags: &ipc.Flags{
			Pwd:     "/tmp",
			VimMode: "insert",
		},
	})
	require.NoError(t, err)

	resp1, err := stream1.Recv()
	require.NoError(t, err)
	// Just check if it contains the vim segment (based on the config used in tests)
	// The test config seems to have a vim segment (from TestDaemonRenderWithVimMode usage)
	// "test-vim" is a config name? No, createTestConfig creates a config.
	// I need to make sure the config includes a vim segment.
	// createTestConfig is defined in daemon_test.go usually. I'll check it later if this fails.
	// Assuming the config has a vim segment that prints the mode.

	// 2. Repaint: NORMAL mode
	stream2, err := client.RenderPrompt(ctx, &ipc.PromptRequest{
		Version:   ipc.ProtocolVersion,
		SessionId: "vim-test-session",
		RequestId: "req-2",
		Repaint:   true,
		Flags: &ipc.Flags{
			Pwd:     "/tmp",
			VimMode: "normal",
		},
	})
	require.NoError(t, err)

	resp2, err := stream2.Recv()
	require.NoError(t, err)

	initialText := resp1.Prompts["primary"].Text
	t.Logf("Initial Prompt Text: %s", initialText)
	assert.Contains(t, initialText, "insert")

	promptText := resp2.Prompts["primary"].Text
	t.Logf("Repaint Prompt Text: %s", promptText)

	assert.NotEqual(t, initialText, promptText, "Prompt text should change on repaint when vim mode changes")
	assert.Contains(t, promptText, "normal")
}
