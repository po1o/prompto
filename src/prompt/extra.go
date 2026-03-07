package prompt

import "github.com/po1o/prompto/src/config"

type ExtraPromptType int

const (
	Transient ExtraPromptType = iota
	Valid
	Error
	Secondary
	Debug
)

func (e *Engine) ExtraPrompt(promptType ExtraPromptType) string {
	e.resetSharedProviders()

	if promptType == Secondary && e.hasLayoutSecondary() {
		return e.renderLayoutExtra(e.LayoutConfig.SecondaryPrompt, false)
	}

	if promptType == Transient && e.hasLayoutTransient() {
		return e.renderLayoutExtra(e.LayoutConfig.TransientPrompt, true)
	}

	return ""
}

func (e *Engine) TransientRPrompt() string {
	e.resetSharedProviders()

	if !e.hasLayoutTransient() || len(e.LayoutConfig.TransientRPrompt) == 0 {
		return ""
	}

	line := e.layoutBlock(&e.LayoutConfig.TransientRPrompt[0], config.RPrompt, config.Right, false)
	text, length := e.writeBlockSegments(line)
	if length == 0 {
		return ""
	}

	return text
}

func (e *Engine) renderLayoutExtra(layouts []config.PromptLayout, finalSpace bool) string {
	didRender := false
	for i := range layouts {
		block := e.layoutBlock(&layouts[i], config.Prompt, config.Left, i != 0)
		cancelNewline := !didRender
		if i == 0 {
			cancelNewline = false
		}

		if e.renderBlock(block, cancelNewline) {
			didRender = true
		}
	}

	if finalSpace && e.Config != nil && e.Config.FinalSpace {
		e.write(" ")
	}

	return e.string()
}
