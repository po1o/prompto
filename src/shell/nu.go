package shell

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed scripts/prompto.nu
var nuInit string

func (f Features) Nu() Code {
	switch f {
	case Transient:
		return `$env.TRANSIENT_PROMPT_COMMAND = {|| _prompto_get_prompt transient }`
	case Upgrade:
		return "^$_prompto_executable upgrade --auto"
	case Notice:
		return "^$_prompto_executable notice"
	case Daemon:
		return enablePromptoDaemon
	case PromptMark, RPrompt, PoshGit, Azure, LineError, Jobs, Tooltips, FTCSMarks, CursorPositioning, Async, VimMode, VimCursorBlink, VimCursorShape:
		fallthrough
	default:
		return ""
	}
}

func quoteNuStr(str string) string {
	if str == "" {
		return "''"
	}

	return fmt.Sprintf(`"%s"`, strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(str))
}
