package shell

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFishFeatures(t *testing.T) {
	got := allFeatures.Lines(FISH).String("// these are the features")

	want := `// these are the features
enable_prompto_tooltips
set --global _prompto_transient_prompt 1
set --global _prompto_ftcs_marks 1
"$_prompto_executable" upgrade --auto
"$_prompto_executable" notice
set --global _prompto_prompt_mark 1
enable_prompto_daemon`

	assert.Equal(t, want, got)
}

func TestFishVimFeatures(t *testing.T) {
	features := VimMode | VimCursorBlink | VimCursorShape

	got := features.Lines(FISH).String("")

	want := `
fish_vi_key_bindings; enable_prompto_vim_mode
set --global _prompto_cursor_blink 1
set --global _prompto_cursor_shape 1; _prompto_should_change_cursor; and _prompto_apply_cursor_shape`

	assert.Equal(t, want, got)
}

func TestQuoteFishStr(t *testing.T) {
	tests := []struct {
		str      string
		expected string
	}{
		{str: "", expected: "''"},
		{str: `/tmp/"omp's dir"/prompto`, expected: `'/tmp/"omp\'s dir"/prompto'`},
		{str: `C:/tmp\omp's dir/prompto.exe`, expected: `'C:/tmp\\omp\'s dir/prompto.exe'`},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.expected, quoteFishStr(tc.str), fmt.Sprintf("quoteFishStr: %s", tc.str))
	}
}
