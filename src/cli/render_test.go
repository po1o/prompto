package cli

import (
	"testing"
	"time"

	"github.com/po1o/prompto/src/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeRenderOutputTextEscapesTransportControlCharacters(t *testing.T) {
	previousPlain := plain
	plain = false
	t.Cleanup(func() {
		plain = previousPlain
	})

	assert.Equal(t, `line1\nline2\\tail\r`, encodeRenderOutputText("line1\nline2\\tail\r"))
}

func TestEncodeRenderOutputTextPreservesPlainOutput(t *testing.T) {
	previousPlain := plain
	plain = true
	t.Cleanup(func() {
		plain = previousPlain
	})

	assert.Equal(t, "line1\nline2", encodeRenderOutputText("line1\nline2"))
}

// TestRenderStreamOutlastsTheDaemonRenderDeadline pins an ordering that spans
// two packages and so can drift without anything noticing. The daemon draws
// segments as timed out at its own deadline; this client has to still be
// listening when that lands, or the marker is produced for nobody and the shell
// keeps the pending placeholders — the failure the marker exists to prevent.
//
// The daemon clamps against the deadline this client sends, so correctness does
// not rest on this. What rests on it is the clamp never having to do anything
// in the default configuration.
func TestRenderStreamOutlastsTheDaemonRenderDeadline(t *testing.T) {
	daemonDefault := (&config.Config{}).GetRenderTimeout()

	require.Greater(t, renderStreamTimeout, daemonDefault,
		"the client must outlast the daemon's own render deadline")
	require.Greater(t, renderStreamTimeout-daemonDefault, 30*time.Second,
		"and by enough that a late segment still gets its marker across")
}
