package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZshFeatures(t *testing.T) {
	got := allFeatures.Lines(ZSH).String("// these are the features")

	want := `// these are the features
enable_prompto_tooltips
_prompto_transient_enabled=1; _prompto_create_widget zle-line-init _prompto_zle-line-init
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

func TestZshDaemonRenderClearsTransientCacheOnNewPrompt(t *testing.T) {
	assert.Contains(t, zshInit, "if [[ $repaint_flag != \"--repaint\" ]]; then")
	assert.Contains(t, zshInit, "_prompto_transient_prompt=")
	assert.Contains(t, zshInit, "_prompto_transient_rprompt=")
}

func TestZshDaemonRenderDrainsBufferedCompletion(t *testing.T) {
	assert.Contains(t, zshInit, "while IFS= read -t 0 -r line <&$fd; do")
	assert.Contains(t, zshInit, "Fast renders can emit the final completion batch right after the initial update batch")
}

func TestZshDaemonRenderGuardsResetPromptOutsideZLE(t *testing.T) {
	assert.Contains(t, zshInit, "function _prompto_reset_prompt_if_zle()")
	assert.Contains(t, zshInit, "if zle; then")
	assert.Contains(t, zshInit, "_prompto_reset_prompt_if_zle")
}
