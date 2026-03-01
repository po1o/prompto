package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZshFeatures(t *testing.T) {
	got := allFeatures.Lines(ZSH).String("// these are the features")

	want := `// these are the features
enable_poshtooltips
_omp_create_widget zle-line-init _omp_zle-line-init
_omp_ftcs_marks=1
"$_omp_executable" upgrade --auto
"$_omp_executable" notice
_omp_cursor_positioning=1
enable_poshdaemon`

	assert.Equal(t, want, got)
}

func TestZshVimFeatures(t *testing.T) {
	features := VimMode | VimCursorBlink | VimCursorShape

	got := features.Lines(ZSH).String("")

	want := `
bindkey -v; _omp_vim_mode=1; _omp_create_widget zle-keymap-select _omp_zle-keymap-select; _omp_setup_vim_keybindings
_omp_cursor_blink=1
_omp_cursor_shape=1; _omp_should_change_cursor && _omp_apply_cursor_shape`

	assert.Equal(t, want, got)
}
