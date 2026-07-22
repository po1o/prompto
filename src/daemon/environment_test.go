package daemon

import (
	"strconv"
	"sync"
	"testing"

	"github.com/po1o/prompto/src/runtime"

	"github.com/stretchr/testify/require"
)

func TestEnvironmentGetenvAuthoritativeClientMap(t *testing.T) {
	t.Setenv("DAEMON_ONLY_VAR", "daemon-value")

	env := NewEnvironment(&runtime.Flags{}, map[string]string{"CLIENT_VAR": "client-value"})

	// Non-nil request map is authoritative: no per-key daemon fallback.
	require.Equal(t, "client-value", env.Getenv("CLIENT_VAR"))
	require.Empty(t, env.Getenv("DAEMON_ONLY_VAR"))

	// Nil request map: fall back to the daemon's own environment.
	env.setEnvVars(nil)
	require.Equal(t, "daemon-value", env.Getenv("DAEMON_ONLY_VAR"))
}

// TestEnvironmentConcurrentGetenvDuringRepaint exercises the envVars swap
// under the race detector: segment goroutines from a previous render
// generation keep calling Getenv while a soft cancel (vim repaint) replaces
// the request env map.
func TestEnvironmentConcurrentGetenvDuringRepaint(t *testing.T) {
	env := NewEnvironment(&runtime.Flags{VimMode: "insert"}, map[string]string{"KEY": "v0"})

	done := make(chan struct{})

	var wg sync.WaitGroup

	for range 4 {
		wg.Go(func() {
			for {
				select {
				case <-done:
					return
				default:
					_ = env.Getenv("KEY")
				}
			}
		})
	}

	for i := range 500 {
		env.UpdateForRepaint(&runtime.Flags{VimMode: "normal"}, map[string]string{"KEY": "v" + strconv.Itoa(i)})
	}

	close(done)
	wg.Wait()

	require.Equal(t, "v499", env.Getenv("KEY"))
}
