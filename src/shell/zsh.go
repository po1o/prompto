package shell

import (
	_ "embed"
)

//go:embed scripts/omp.zsh
var zshInit string

func (f Features) Zsh() Code {
	switch f {
	case CursorPositioning:
		return unixCursorPositioning
	case Tooltips:
		return "enable_poshtooltips"
	case Transient:
		return "_omp_create_widget zle-line-init _omp_zle-line-init"
	case FTCSMarks:
		return unixFTCSMarks
	case Upgrade:
		return unixUpgrade
	case Notice:
		return unixNotice
	case Daemon:
		return enablePoshDaemon
	case VimMode:
		return "bindkey -v; _omp_vim_mode=1; _omp_create_widget zle-keymap-select _omp_zle-keymap-select; _omp_setup_vim_keybindings"
	case VimCursorBlink:
		return "_omp_cursor_blink=1"
	case VimCursorShape:
		return "_omp_cursor_shape=1; _omp_should_change_cursor && _omp_apply_cursor_shape"
	case PromptMark, RPrompt, PoshGit, Azure, LineError, Jobs, Async:
		fallthrough
	default:
		return ""
	}
}
