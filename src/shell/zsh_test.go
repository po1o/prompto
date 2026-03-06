package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZshFeatures(t *testing.T) {
	got := allFeatures.Lines(ZSH).String("// these are the features")

	want := `// these are the features
enable_prompto_tooltips
_prompto_create_widget zle-line-init _prompto_zle-line-init
_prompto_ftcs_marks=1
"$_prompto_executable" upgrade --auto
"$_prompto_executable" notice
_prompto_cursor_positioning=1
enable_prompto_daemon`

	assert.Equal(t, want, got)
}

func TestZshVimFeatures(t *testing.T) {
	features := VimMode | VimCursorBlink | VimCursorShape

	got := features.Lines(ZSH).String("")

	want := `
bindkey -v; _prompto_vim_mode=1; _prompto_create_widget zle-keymap-select _prompto_zle-keymap-select; _prompto_setup_vim_keybindings
_prompto_cursor_blink=1
_prompto_cursor_shape=1; _prompto_should_change_cursor && _prompto_apply_cursor_shape`

	assert.Equal(t, want, got)
}
