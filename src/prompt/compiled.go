package prompt

import "github.com/po1o/prompto/src/config"

func (e *Engine) hasCompiledPrimaryLayout() bool {
	if e.CompiledConfig == nil {
		return false
	}

	return len(e.CompiledConfig.Prompt) > 0 || len(e.CompiledConfig.RPrompt) > 0
}

func (e *Engine) hasCompiledSecondaryLayout() bool {
	if e.CompiledConfig == nil {
		return false
	}

	return len(e.CompiledConfig.SecondaryPrompt) > 0
}

func (e *Engine) hasCompiledTransientLayout() bool {
	if e.CompiledConfig == nil {
		return false
	}

	return len(e.CompiledConfig.TransientPrompt) > 0 || len(e.CompiledConfig.TransientRPrompt) > 0
}

func (e *Engine) compiledLayoutBlock(layout *config.PromptLayout, blockType config.BlockType, alignment config.BlockAlignment, newline bool) *config.Block {
	block := &config.Block{
		Type:            blockType,
		Alignment:       alignment,
		Filler:          layout.Filler,
		LeadingDiamond:  layout.LeadingDiamond,
		TrailingDiamond: layout.TrailingDiamond,
		Newline:         newline,
	}

	for _, name := range layout.Segments {
		segmentDef, ok := e.CompiledConfig.Segments[name]
		if !ok {
			continue
		}

		segment := segmentDef.Clone()
		if alignment == config.Right {
			orientedLeading := mirrorSeparator(segment.TrailingDiamond)
			orientedTrailing := mirrorSeparator(segment.LeadingDiamond)
			segment.LeadingDiamond = orientedLeading
			segment.TrailingDiamond = orientedTrailing
		}

		block.Segments = append(block.Segments, segment)
	}

	return block
}

func (e *Engine) compiledPrimaryBlocks() []*config.Block {
	if e.CompiledConfig == nil {
		return nil
	}

	lineCount := max(len(e.CompiledConfig.RPrompt), len(e.CompiledConfig.Prompt))
	blocks := make([]*config.Block, 0, lineCount*2)

	for i := range lineCount {
		if i < len(e.CompiledConfig.Prompt) {
			left := e.compiledLayoutBlock(&e.CompiledConfig.Prompt[i], config.Prompt, config.Left, i != 0)
			blocks = append(blocks, left)
		}

		if i < len(e.CompiledConfig.RPrompt) {
			right := e.compiledLayoutBlock(&e.CompiledConfig.RPrompt[i], config.RPrompt, config.Right, false)
			if i < len(e.CompiledConfig.Prompt) {
				right.Filler = e.CompiledConfig.Prompt[i].Filler
			}

			blocks = append(blocks, right)
		}
	}

	return blocks
}

func mirrorSeparator(input string) string {
	switch input {
	case "\uE0B0":
		return "\uE0B2"
	case "\uE0B2":
		return "\uE0B0"
	case "\uE0B1":
		return "\uE0B3"
	case "\uE0B3":
		return "\uE0B1"
	case "\uE0B4":
		return "\uE0B6"
	case "\uE0B6":
		return "\uE0B4"
	case "\uE0BA":
		return "\uE0BC"
	case "\uE0BC":
		return "\uE0BA"
	case "\uE0BE":
		return "\uE0B8"
	case "\uE0B8":
		return "\uE0BE"
	case "\uE0C0":
		return "\uE0C1"
	case "\uE0C1":
		return "\uE0C0"
	case "\uE0CE":
		return "\uE0CF"
	case "\uE0CF":
		return "\uE0CE"
	default:
		return input
	}
}
