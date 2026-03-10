package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
