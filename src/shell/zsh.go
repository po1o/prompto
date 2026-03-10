package shell

import (
	_ "embed"
)

//go:embed scripts/prompto.zsh
var zshInit string

func (f Features) Zsh() Code {
	switch f {
	case CursorPositioning:
		return unixCursorPositioning
	case Tooltips:
		return "enable_prompto_tooltips"
	case Transient:
		return "_prompto_transient_enabled=1; _prompto_create_widget zle-line-init _prompto_zle-line-init"
	case FTCSMarks:
		return unixFTCSMarks
	case Daemon:
		return enablePromptoDaemon
	case VimMode:
		return "bindkey -v; _prompto_vim_mode=1; _prompto_create_widget zle-keymap-select _prompto_zle-keymap-select; _prompto_setup_vim_keybindings"
	case VimCursorBlink:
		return "_prompto_cursor_blink=1"
	case VimCursorShape:
		return "_prompto_cursor_shape=1; _prompto_should_change_cursor && _prompto_apply_cursor_shape"
	case PromptMark, RPrompt, PoshGit, Azure, LineError, Jobs, Async:
		fallthrough
	default:
		return ""
	}
}
