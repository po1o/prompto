package prompt

import (
	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/runtime"
	"github.com/po1o/prompto/src/shell"
)

const (
	RPromptKey       = "rprompt"
	RPromptLengthKey = "rprompt_length"
)

func (e *Engine) RPrompt() string {
	e.resetSharedProviders()
	if e.LayoutConfig == nil || len(e.LayoutConfig.RPrompt) == 0 {
		return ""
	}

	if e.shouldInlinePrimaryRPrompt() {
		text, length := e.renderLayoutRightPrompt(e.LayoutConfig.RPrompt[len(e.LayoutConfig.RPrompt)-1:])
		if length == 0 {
			return ""
		}

		e.rpromptLength = length
		return text
	}

	text, length := e.renderLayoutRightPrompt(e.LayoutConfig.RPrompt)

	// do not print anything when we don't have any text
	if length == 0 {
		return ""
	}

	e.rpromptLength = length

	if e.Env.Shell() == shell.ELVISH && e.Env.GOOS() != runtime.WINDOWS {
		// Workaround to align with a right-aligned block on non-Windows systems.
		text += " "
	}

	if !e.Config.ToolTipsAction.IsDefault() {
		cache.Set(cache.Session, RPromptKey, text, cache.INFINITE)
		cache.Set(cache.Session, RPromptLengthKey, e.rpromptLength, cache.INFINITE)
	}

	return text
}

func (e *Engine) renderLayoutRightPrompt(layouts []config.PromptLayout) (string, int) {
	e.rprompt = ""
	e.rpromptLength = 0

	for i := range layouts {
		line := e.layoutBlock(&layouts[i], config.RPrompt, config.Right, i != 0)
		text, length := e.writeBlockSegments(line)
		e.appendRightPromptLine(text, length, i != 0)
	}

	return e.rprompt, e.rpromptLength
}
