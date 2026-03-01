package prompt

import "github.com/jandedobbeleer/oh-my-posh/src/config"

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

		block.Segments = append(block.Segments, segmentDef.Clone())
	}

	return block
}
