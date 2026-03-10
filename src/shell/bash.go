package shell

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed scripts/prompto.bash
var bashInit string

func (f Features) Bash() Code {
	switch f {
	case CursorPositioning:
		return unixCursorPositioning
	case FTCSMarks:
		return unixFTCSMarks
	case Daemon:
		return enablePromptoDaemon
	case VimMode:
		return "set -o vi; enable_prompto_vim_mode"
	case VimCursorBlink:
		return "_prompto_cursor_blink=1"
	case VimCursorShape:
		return "_prompto_cursor_shape=1; _prompto_should_change_cursor && _prompto_apply_cursor_shape"
	case RPrompt:
		if !bashBLEsession {
			return ""
		}

		return `bleopt prompt_rps1='$(
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
)'`
	case Transient:
		if !bashBLEsession {
			return ""
		}

		return `_prompto_transient_enabled=1
bleopt prompt_ps1_transient=always
bleopt prompt_ps1_final='$(
    "$_prompto_executable" render transient \
        --shell=bash \
        --shell-version="$BASH_VERSION" \
        --escape=false
)'`
	case PromptMark, PoshGit, Azure, LineError, Jobs, Tooltips, Async:
		fallthrough
	default:
		return ""
	}
}

func QuotePosixStr(str string) string {
	if str == "" {
		return "''"
	}

	return fmt.Sprintf("$'%s'", strings.NewReplacer(`\`, `\\`, "'", `\'`).Replace(str))
}
