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
		return e.renderLayoutExtra(e.LayoutConfig.SecondaryPrompt)
	}

	if promptType == Transient && e.hasLayoutTransient() {
		return e.renderLayoutExtra(e.LayoutConfig.TransientPrompt)
	}

	return ""
}

func (e *Engine) renderLayoutExtra(layouts []config.PromptLayout) string {
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

	return e.string()
}
