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

	line := e.layoutBlock(&e.LayoutConfig.RPrompt[0], config.RPrompt, config.Right, false)
	text, length := e.writeBlockSegments(line)

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
