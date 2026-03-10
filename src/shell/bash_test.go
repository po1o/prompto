package shell

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBashFeatures(t *testing.T) {
	got := allFeatures.Lines(BASH).String("// these are the features")

	want := `// these are the features
_prompto_ftcs_marks=1
_prompto_cursor_positioning=1
enable_prompto_daemon`

	assert.Equal(t, want, got)
}

func TestBashFeaturesWithBLE(t *testing.T) {
	bashBLEsession = true

	got := allFeatures.Lines(BASH).String("// these are the features")

	want := `// these are the features
_prompto_transient_enabled=1
bleopt prompt_ps1_transient=always
bleopt prompt_ps1_final='$(
    "$_prompto_executable" render transient \
        --shell=bash \
        --shell-version="$BASH_VERSION" \
        --escape=false
)'
_prompto_ftcs_marks=1
bleopt prompt_rps1='$(
	"$_prompto_executable" render right \
		--shell=bash \
		--shell-version="$BASH_VERSION" \
		--status="$_prompto_status" \
		--pipestatus="${_prompto_pipestatus[*]}" \
		--no-status="$_prompto_no_status" \
		--execution-time="$_prompto_execution_time" \
		--stack-count="$_prompto_stack_count" \
		--terminal-width="${COLUMNS-0}" \
		--escape=false
)'
_prompto_cursor_positioning=1
enable_prompto_daemon`

	assert.Equal(t, want, got)

	bashBLEsession = false
}

func TestBashVimFeatures(t *testing.T) {
	features := VimMode | VimCursorBlink | VimCursorShape

	got := features.Lines(BASH).String("")

	want := `
set -o vi; enable_prompto_vim_mode
_prompto_cursor_blink=1
_prompto_cursor_shape=1; _prompto_should_change_cursor && _prompto_apply_cursor_shape`

	assert.Equal(t, want, got)
}

func TestBashInitDecodesEscapedRenderOutput(t *testing.T) {
	assert.Contains(t, bashInit, "function _prompto_decode_render_text()")
	assert.Contains(t, bashInit, "_prompto_decode_render_text \"${line#*:}\"")
}

func TestQuotePosixStr(t *testing.T) {
	tests := []struct {
		str      string
		expected string
	}{
		{str: "", expected: "''"},
		{str: `/tmp/"omp's dir"/prompto`, expected: `$'/tmp/"omp\'s dir"/prompto'`},
		{str: `C:/tmp\omp's dir/prompto.exe`, expected: `$'C:/tmp\\omp\'s dir/prompto.exe'`},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.expected, QuotePosixStr(tc.str), fmt.Sprintf("QuotePosixStr: %s", tc.str))
	}
}
