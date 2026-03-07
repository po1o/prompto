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
	return e.ExtraPromptNoReset(promptType)
}

// ExtraPromptNoReset renders an extra prompt while reusing the current shared-provider scope.
func (e *Engine) ExtraPromptNoReset(promptType ExtraPromptType) string {
	switch promptType {
	case Secondary:
		if e.hasLayoutSecondary() {
			return e.renderLayoutExtra(e.LayoutConfig.SecondaryPrompt, false)
		}
	case Transient:
		if e.hasLayoutTransient() {
			return e.renderLayoutExtra(e.LayoutConfig.TransientPrompt, true)
		}
	case Valid:
		return e.renderSingleExtraSegment(e.Config.ValidLine)
	case Error:
		return e.renderSingleExtraSegment(e.Config.ErrorLine)
	case Debug:
		return e.renderSingleExtraSegment(e.Config.DebugPrompt)
	}

	return ""
}

func (e *Engine) TransientRPrompt() string {
	e.resetSharedProviders()
	return e.TransientRPromptNoReset()
}

// TransientRPromptNoReset renders the transient right prompt while reusing the current shared-provider scope.
func (e *Engine) TransientRPromptNoReset() string {
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

func (e *Engine) renderSingleExtraSegment(segment *config.Segment) string {
	if segment == nil {
		return ""
	}

	cloned := segment.Clone()
	if cloned.Type == "" {
		cloned.Type = config.TEXT
	}

	block := &config.Block{
		Type:      config.Prompt,
		Alignment: config.Left,
		Segments:  []*config.Segment{cloned},
	}

	text, length := e.writeBlockSegments(block)
	if length == 0 {
		return ""
	}

	return text
}
