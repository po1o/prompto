package cli

import (
	"testing"
	"time"

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

// TestRenderStreamOutlastsASlowSegment pins the streaming render's deadline
// apart from the single-call one. They were the same ten seconds, so a segment
// slower than that — a git status on a large repository takes twelve — could
// never deliver its update: the client abandoned the stream first, the shell
// kept the pending placeholders it had already drawn, and the next prompt
// cancelled the render, killing the command just short of finishing. The
// segment resolved on no prompt, ever.
//
// This pins the intent rather than the exact value. The streaming deadline is
// a safety net against a daemon that has stopped answering, so it has to
// outlast any segment worth waiting for.
func TestRenderStreamOutlastsASlowSegment(t *testing.T) {
	require.Greater(t, renderStreamTimeout, daemonCallTimeout,
		"a streaming render must not be held to the timeout for a single call")

	const slowestRealisticSegment = 30 * time.Second
	require.GreaterOrEqual(t, renderStreamTimeout, slowestRealisticSegment,
		"a segment slower than this loses its update entirely, it does not degrade")
}
