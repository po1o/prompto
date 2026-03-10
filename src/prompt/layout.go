package prompt

import "github.com/po1o/prompto/src/config"

func (e *Engine) hasLayoutPrimary() bool {
	if e.LayoutConfig == nil {
		return false
	}

	return len(e.LayoutConfig.Prompt) > 0 || len(e.LayoutConfig.RPrompt) > 0
}

func (e *Engine) hasLayoutSecondary() bool {
	if e.LayoutConfig == nil {
		return false
	}

	return len(e.LayoutConfig.SecondaryPrompt) > 0
}

func (e *Engine) hasLayoutTransient() bool {
	if e.LayoutConfig == nil {
		return false
	}

	return len(e.LayoutConfig.TransientPrompt) > 0 || len(e.LayoutConfig.TransientRPrompt) > 0
}

func shouldInlineRightLayout(leftCount, rightCount int) bool {
	if rightCount == 0 {
		return false
	}

	return leftCount > 1 || rightCount > 1
}

func (e *Engine) shouldInlinePrimaryRPrompt() bool {
	if e.LayoutConfig == nil {
		return false
	}

	return shouldInlineRightLayout(len(e.LayoutConfig.Prompt), len(e.LayoutConfig.RPrompt))
}

func (e *Engine) shouldInlineTransientRPrompt() bool {
	if e.LayoutConfig == nil {
		return false
	}

	return shouldInlineRightLayout(len(e.LayoutConfig.TransientPrompt), len(e.LayoutConfig.TransientRPrompt))
}

func (e *Engine) layoutBlock(layout *config.PromptLayout, blockType config.BlockType, alignment config.BlockAlignment, newline bool) *config.Block {
	block := &config.Block{
		Type:            blockType,
		Alignment:       alignment,
		Filler:          layout.Filler,
		LeadingDiamond:  layout.LeadingDiamond,
		TrailingDiamond: layout.TrailingDiamond,
		Newline:         newline,
	}

	for _, name := range layout.Segments {
		segmentDef, ok := e.LayoutConfig.Segments[name]
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

func (e *Engine) composeLayoutBlocks(leftLayouts, rightLayouts []config.PromptLayout, inlineRight bool) []*config.Block {
	lineCount := max(len(rightLayouts), len(leftLayouts))
	blocks := make([]*config.Block, 0, lineCount*2)

	for i := range lineCount {
		hasLeft := i < len(leftLayouts)
		hasRight := i < len(rightLayouts)
		keepRightForRPrompt := inlineRight && hasRight && i == len(rightLayouts)-1

		if hasLeft {
			left := e.layoutBlock(&leftLayouts[i], config.Prompt, config.Left, i != 0)
			blocks = append(blocks, left)
		}

		if keepRightForRPrompt && !hasLeft {
			blocks = append(blocks, &config.Block{
				Type:      config.Prompt,
				Alignment: config.Left,
				Newline:   i != 0,
				Force:     true,
			})
		}

		if !hasRight {
			continue
		}

		rightType := config.RPrompt
		rightNewline := false
		if inlineRight && !keepRightForRPrompt {
			rightType = config.Prompt
			rightNewline = i != 0 && i >= len(leftLayouts)
		}

		right := e.layoutBlock(&rightLayouts[i], rightType, config.Right, rightNewline)
		if hasLeft {
			right.Filler = leftLayouts[i].Filler
		}

		blocks = append(blocks, right)
	}

	return blocks
}

func (e *Engine) layoutPrimaryBlocks() []*config.Block {
	if e.LayoutConfig == nil {
		return nil
	}

	return e.composeLayoutBlocks(e.LayoutConfig.Prompt, e.LayoutConfig.RPrompt, e.shouldInlinePrimaryRPrompt())
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
