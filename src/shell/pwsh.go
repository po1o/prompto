package shell

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed scripts/prompto.ps1
var pwshInit string

// Pwsh returns the snippet of PowerShell code that activates the given
// feature when emitted into the rendered init script. Script body lives in
// scripts/prompto.ps1. Vim-mode features assume PSReadLine is configured
// with `-EditMode Vi`.
func (f Features) Pwsh() Code {
	switch f {
	case Tooltips:
		return "Enable-PromptoTooltips"
	case LineError:
		return "Enable-PromptoLineError"
	case Transient:
		return "Enable-PromptoTransientPrompt"
	case Jobs:
		return "$global:_promptoJobCount = $true"
	case Azure:
		return "$global:_promptoAzure = $true"
	case PoshGit:
		return "$global:_promptoPoshGit = $true"
	case FTCSMarks:
		return "$global:_promptoFTCSMarks = $true"
	case Daemon:
		return "Enable-PromptoDaemon"
	case VimMode:
		return "Set-PSReadLineOption -EditMode Vi; Enable-PromptoVimMode"
	case VimCursorBlink:
		return "$script:CursorBlink = $true"
	case VimCursorShape:
		return "$script:CursorShape = $true; Set-VimModeCursorFromState"
	case PromptMark, RPrompt, CursorPositioning, Async:
		fallthrough
	default:
		return ""
	}
}

func quotePwshOrElvishStr(str string) string {
	if str == "" {
		return "''"
	}

	return fmt.Sprintf("'%s'", strings.ReplaceAll(str, "'", "''"))
}
