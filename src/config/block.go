package config

// BlockType type of block
type BlockType string

// BlockAlignment alignment of a Block
type BlockAlignment string

// Overflow defines how to handle a right block that overflows with the previous block
type Overflow string

const (
	// Prompt writes one or more Segments
	Prompt BlockType = "prompt"
	// RPrompt is a right aligned prompt
	RPrompt BlockType = "rprompt"
	// Left aligns left
	Left BlockAlignment = "left"
	// Right aligns right
	Right BlockAlignment = "right"
	// Break adds a line break
	Break Overflow = "break"
	// Hide hides the block
	Hide Overflow = "hide"
)

// Block defines a part of the prompt with optional segments
type Block struct {
	Type            BlockType      `yaml:"type,omitempty"`
	Alignment       BlockAlignment `yaml:"alignment,omitempty"`
	Filler          string         `yaml:"filler,omitempty"`
	Overflow        Overflow       `yaml:"overflow,omitempty"`
	LeadingDiamond  string         `yaml:"leading_diamond,omitempty"`
	TrailingDiamond string         `yaml:"trailing_diamond,omitempty"`
	Segments        []*Segment     `yaml:"segments,omitempty"`
	Index           int            `yaml:"index,omitempty"`
	Newline         bool           `yaml:"newline,omitempty"`
	Force           bool           `yaml:"force,omitempty"`
}
