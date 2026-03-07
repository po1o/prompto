package prompt

import (
	"fmt"
	"strings"

	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/shell"
	"github.com/po1o/prompto/src/terminal"
)

func (e *Engine) Primary() string {
	e.resetSharedProviders()

	needsPrimaryRightPrompt := false

	e.writePrimaryPrompt(needsPrimaryRightPrompt)

	switch e.Env.Shell() {
	case shell.ZSH:
		if !e.Env.Flags().Eval {
			break
		}

		// Warp doesn't support RPROMPT so we need to write it manually
		if e.isWarp() {
			e.writePrimaryRightPrompt()
			prompt := fmt.Sprintf("PS1=%s", shell.QuotePosixStr(e.string()))
			return prompt
		}

		prompt := fmt.Sprintf("PS1=%s", shell.QuotePosixStr(e.string()))
		prompt += fmt.Sprintf("\nRPROMPT=%s", shell.QuotePosixStr(e.rprompt))

		return prompt
	default:
		if !needsPrimaryRightPrompt {
			break
		}

		e.writePrimaryRightPrompt()
	}

	return e.string()
}

func (e *Engine) writePrimaryPrompt(needsPrimaryRPrompt bool) {
	_ = needsPrimaryRPrompt
	e.writeLayoutPrimaryPrompt()
}

func (e *Engine) writeLayoutPrimaryPrompt() {
	if e.Config.ShellIntegration {
		exitCode, _ := e.Env.StatusCodes()
		e.write(terminal.CommandFinished(exitCode, e.Env.Flags().NoExitCode))
		e.write(terminal.PromptStart())
	}

	cycle = &e.Config.Cycle
	var cancelNewline, didRender bool

	lineCount := max(len(e.LayoutConfig.RPrompt), len(e.LayoutConfig.Prompt))

	for i := range lineCount {
		if i == 0 {
			row, _ := e.Env.CursorPosition()
			cancelNewline = e.Env.Flags().Cleared || e.Env.Flags().PromptCount == 1 || row == 1
		}

		if i != 0 {
			cancelNewline = !didRender
		}

		if i < len(e.LayoutConfig.Prompt) {
			left := e.layoutBlock(&e.LayoutConfig.Prompt[i], config.Prompt, config.Left, i != 0)
			if e.renderBlock(left, cancelNewline) {
				didRender = true
			}
		}

		if i < len(e.LayoutConfig.RPrompt) {
			right := e.layoutBlock(&e.LayoutConfig.RPrompt[i], config.RPrompt, config.Right, false)
			if i < len(e.LayoutConfig.Prompt) {
				right.Filler = e.LayoutConfig.Prompt[i].Filler
			}

			if e.renderBlock(right, true) {
				didRender = true
			}
		}
	}

	if len(e.Config.ConsoleTitleTemplate) > 0 && !e.Env.Flags().Plain {
		title := e.getTitleTemplateText()
		e.write(terminal.FormatTitle(title))
	}

	if e.Config.FinalSpace {
		e.write(" ")
		e.currentLineLength++
	}

	if e.Config.ITermFeatures != nil && e.isIterm() {
		host, _ := e.Env.Host()
		e.write(terminal.RenderItermFeatures(e.Config.ITermFeatures, e.Env.Shell(), e.Env.Pwd(), e.Env.User(), host))
	}

	if e.Config.ShellIntegration {
		e.write(terminal.CommandStart())
	}

	e.pwd()
}

func (e *Engine) needsPrimaryRightPrompt() bool {
	if e.Env.Flags().Debug {
		return true
	}

	switch e.Env.Shell() {
	case shell.PWSH, shell.GENERIC, shell.ZSH:
		return true
	default:
		return false
	}
}

func (e *Engine) writePrimaryRightPrompt() {
	space, OK := e.canWriteRightBlock(e.rpromptLength, true)
	if !OK {
		return
	}

	e.write(terminal.SaveCursorPosition())
	e.write(strings.Repeat(" ", space))
	e.write(e.rprompt)
	e.write(terminal.RestoreCursorPosition())
}
