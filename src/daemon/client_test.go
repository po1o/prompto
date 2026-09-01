package daemon

import (
	"errors"
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
