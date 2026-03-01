package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCompiledTOMLWithTypedInstances(t *testing.T) {
	raw := `
[[prompt]]
segments = ["session", "path"]
filler = " "
leading_style = "powerline"
trailing_style = "powerline"

[[rprompt]]
segments = ["git.main"]
leading_style = "powerline"
trailing_style = "powerline"

[[secondary_prompt]]
segments = ["path"]

[[transient_prompt]]
segments = ["session"]

[[transient_rprompt]]
segments = ["git.main"]

[session]
type = "session"
style = "plain"

[path]
style = "powerline"
leading_style = "rounded"
trailing_separator = ">"

[git.main]
style = "powerline"
[git.main.options]
branch_max_length = 20
`

	cfg, err := ParseCompiledTOML([]byte(raw))
	require.NoError(t, err)

	require.Len(t, cfg.Prompt, 1)
	assert.Equal(t, []string{"session", "path"}, cfg.Prompt[0].Segments)
	assert.Equal(t, " ", cfg.Prompt[0].Filler)
	assert.Equal(t, "\uE0B2", cfg.Prompt[0].LeadingDiamond)
	assert.Equal(t, "\uE0B0", cfg.Prompt[0].TrailingDiamond)
	assert.Empty(t, cfg.Prompt[0].LeadingStyle)
	assert.Empty(t, cfg.Prompt[0].TrailingStyle)
	assert.Empty(t, cfg.Prompt[0].LeadingSeparator)
	assert.Empty(t, cfg.Prompt[0].TrailingSeparator)

	require.Len(t, cfg.RPrompt, 1)
	assert.Equal(t, []string{"git.main"}, cfg.RPrompt[0].Segments)
	assert.Equal(t, "\uE0B0", cfg.RPrompt[0].LeadingDiamond)
	assert.Equal(t, "\uE0B2", cfg.RPrompt[0].TrailingDiamond)
	assert.Empty(t, cfg.RPrompt[0].LeadingStyle)
	assert.Empty(t, cfg.RPrompt[0].TrailingStyle)
	assert.Empty(t, cfg.RPrompt[0].LeadingSeparator)
	assert.Empty(t, cfg.RPrompt[0].TrailingSeparator)
	require.Len(t, cfg.SecondaryPrompt, 1)
	require.Len(t, cfg.TransientPrompt, 1)
	require.Len(t, cfg.TransientRPrompt, 1)

	require.Len(t, cfg.Segments, 3)
	assert.Equal(t, SESSION, cfg.Segments["session"].Type)
	assert.Equal(t, "session", cfg.Segments["session"].Alias)
	assert.Equal(t, PATH, cfg.Segments["path"].Type)
	assert.Equal(t, "path", cfg.Segments["path"].Alias)
	assert.Equal(t, "\uE0B6", cfg.Segments["path"].LeadingDiamond)
	assert.Equal(t, ">", cfg.Segments["path"].TrailingDiamond)
	assert.Equal(t, GIT, cfg.Segments["git.main"].Type)
	assert.Equal(t, "git.main", cfg.Segments["git.main"].Alias)
	assert.Equal(t, float64(20), cfg.Segments["git.main"].Options["branch_max_length"])
}

func TestParseCompiledTOMLReturnsErrorForUnknownSegmentReference(t *testing.T) {
	raw := `
[[prompt]]
segments = ["missing"]

[session]
type = "session"
`

	_, err := ParseCompiledTOML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "unknown segment")
}

func TestParseCompiledTOMLReturnsErrorForInvalidSegmentType(t *testing.T) {
	raw := `
[[prompt]]
segments = ["custom"]

[custom]
type = "not-real"
`

	_, err := ParseCompiledTOML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "unsupported segment type")
}

func TestParseCompiledTOMLReturnsErrorWhenTypeCannotBeInferred(t *testing.T) {
	raw := `
[[prompt]]
segments = ["main"]

[main]
style = "powerline"
`

	_, err := ParseCompiledTOML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "missing type")
}

func TestParseCompiledTOMLReturnsErrorForDirectPromptDiamonds(t *testing.T) {
	raw := `
[[prompt]]
segments = ["session"]
leading_diamond = "<"

[session]
type = "session"
`

	_, err := ParseCompiledTOML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "does not allow leading_diamond/trailing_diamond")
}

func TestParseCompiledTOMLReturnsErrorForMutuallyExclusiveLineSeparatorConfig(t *testing.T) {
	raw := `
[[prompt]]
segments = ["session"]
leading_style = "powerline"
leading_separator = "<"

[session]
type = "session"
`

	_, err := ParseCompiledTOML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "cannot define both leading_style and leading_separator")
}

func TestParseCompiledTOMLReturnsErrorForDirectSegmentDiamonds(t *testing.T) {
	raw := `
[[prompt]]
segments = ["session"]

[session]
type = "session"
leading_diamond = "<"
`

	_, err := ParseCompiledTOML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "does not allow leading_diamond")
}

func TestParseCompiledTOMLReturnsErrorForMutuallyExclusiveSegmentSeparatorConfig(t *testing.T) {
	raw := `
[[prompt]]
segments = ["session"]

[session]
type = "session"
leading_style = "powerline"
leading_separator = "<"
`

	_, err := ParseCompiledTOML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "cannot define both leading_style and leading_separator")
}
