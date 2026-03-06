package shell

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed scripts/prompto.fish
var fishInit string

func (f Features) Fish() Code {
	switch f {
	case Transient:
		return "set --global _prompto_transient_prompt 1"
	case FTCSMarks:
		return "set --global _prompto_ftcs_marks 1"
	case PromptMark:
		return "set --global _prompto_prompt_mark 1"
	case Tooltips:
		return "enable_prompto_tooltips"
	case Upgrade:
		return unixUpgrade
	case Notice:
		return unixNotice
	case Daemon:
		return enablePromptoDaemon
	case VimMode:
		return "fish_vi_key_bindings; enable_prompto_vim_mode"
	case VimCursorBlink:
		return "set --global _prompto_cursor_blink 1"
	case VimCursorShape:
		return "set --global _prompto_cursor_shape 1; _prompto_should_change_cursor; and _prompto_apply_cursor_shape"
	case RPrompt, PoshGit, Azure, LineError, Jobs, CursorPositioning, Async:
		fallthrough
	default:
		return ""
	}
}

func quoteFishStr(str string) string {
	if str == "" {
		return "''"
	}

	return fmt.Sprintf("'%s'", strings.NewReplacer(`\`, `\\`, "'", `\'`).Replace(str))
}
