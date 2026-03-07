package config

import (
	"testing"

	"github.com/po1o/prompto/src/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLayoutYAMLWithTypedInstances(t *testing.T) {
	raw := `
prompt:
  - segments: ["session", "path"]
    filler: " "
    leading_style: "powerline"
    trailing_style: "powerline"

rprompt:
  - segments: ["git.main"]
    leading_style: "powerline"
    trailing_style: "powerline"

secondary:
  - segments: ["path"]

transient:
  - segments: ["session"]

rtransient:
  - segments: ["git.main"]

session:
  type: "session"
  style: "plain"

path:
  leading_style: "rounded"
  trailing_separator: ">"

git.main:
  style: "powerline"
  options:
    branch_max_length: 20
`

	cfg, err := ParseLayoutYAML([]byte(raw))
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
	assert.Equal(t, "\uE0B2", cfg.RPrompt[0].LeadingDiamond)
	assert.Equal(t, "\uE0B0", cfg.RPrompt[0].TrailingDiamond)
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

func TestParseLayoutYAMLStyleShortcutOnPromptLines(t *testing.T) {
	raw := `
prompt:
  - segments: ["session"]
    style: "powerline"

rprompt:
  - segments: ["session"]
    style: "powerline"

session:
  type: "session"
`

	cfg, err := ParseLayoutYAML([]byte(raw))
	require.NoError(t, err)

	require.Len(t, cfg.Prompt, 1)
	assert.Equal(t, "", cfg.Prompt[0].LeadingDiamond)
	assert.Equal(t, "\uE0B0", cfg.Prompt[0].TrailingDiamond)

	require.Len(t, cfg.RPrompt, 1)
	assert.Equal(t, "\uE0B2", cfg.RPrompt[0].LeadingDiamond)
	assert.Equal(t, "", cfg.RPrompt[0].TrailingDiamond)
}

func TestParseLayoutYAMLStyleShortcutOnSegments(t *testing.T) {
	raw := `
prompt:
  - segments: ["git"]

git:
  style: "powerline"
`

	cfg, err := ParseLayoutYAML([]byte(raw))
	require.NoError(t, err)

	segment := cfg.Segments["git"]
	require.NotNil(t, segment)
	assert.Equal(t, Diamond, segment.Style)
	assert.Equal(t, "", segment.LeadingDiamond)
	assert.Equal(t, "\uE0B0", segment.TrailingDiamond)
}

func TestParseLayoutYAMLReturnsErrorForUnknownSegmentReference(t *testing.T) {
	raw := `
prompt:
  - segments: ["missing"]

session:
  type: "session"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "unknown segment")
}

func TestParseLayoutYAMLReturnsErrorForInvalidSegmentType(t *testing.T) {
	raw := `
prompt:
  - segments: ["custom"]

custom:
  type: "not-real"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "unsupported segment type")
}

func TestParseLayoutYAMLReturnsErrorWhenTypeCannotBeInferred(t *testing.T) {
	raw := `
prompt:
  - segments: ["main"]

main:
  style: "powerline"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "missing type")
}

func TestParseLayoutYAMLReturnsErrorForDirectPromptDiamonds(t *testing.T) {
	raw := `
prompt:
  - segments: ["session"]
    leading_diamond: "<"

session:
  type: "session"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "does not allow leading_diamond/trailing_diamond")
}

func TestParseLayoutYAMLReturnsErrorForMutuallyExclusiveLineSeparatorConfig(t *testing.T) {
	raw := `
prompt:
  - segments: ["session"]
    leading_style: "powerline"
    leading_separator: "<"

session:
  type: "session"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "cannot define both leading_style and leading_separator")
}

func TestParseLayoutYAMLReturnsErrorForStyleShortcutMixedWithLineOverrides(t *testing.T) {
	raw := `
prompt:
  - segments: ["session"]
    style: "powerline"
    trailing_style: "rounded"

session:
  type: "session"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "cannot define style together with explicit leading/trailing separator settings")
}

func TestParseLayoutYAMLReturnsErrorForDirectSegmentDiamonds(t *testing.T) {
	raw := `
prompt:
  - segments: ["session"]

session:
  type: "session"
  leading_diamond: "<"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "does not allow leading_diamond")
}

func TestParseLayoutYAMLReturnsErrorForMutuallyExclusiveSegmentSeparatorConfig(t *testing.T) {
	raw := `
prompt:
  - segments: ["session"]

session:
  type: "session"
  leading_style: "powerline"
  leading_separator: "<"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "cannot define both leading_style and leading_separator")
}

func TestParseLayoutYAMLReturnsErrorForStyleShortcutMixedWithSegmentOverrides(t *testing.T) {
	raw := `
prompt:
  - segments: ["session"]

session:
  style: "powerline"
  leading_style: "rounded"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "cannot define style together with explicit leading/trailing separator settings")
}

func TestParseLayoutYAMLAllowsVimModeTopLevelConfig(t *testing.T) {
	raw := `
vim-mode:
  enabled: true
  cursor_shape: true

prompt:
  - segments: ["session"]

session:
  type: "session"
`

	cfg, err := ParseLayoutYAML([]byte(raw))
	require.NoError(t, err)
	require.Len(t, cfg.Prompt, 1)
	require.Contains(t, cfg.Segments, "session")
}

func TestParseLayoutYAMLInfersVimSegmentType(t *testing.T) {
	raw := `
vim-mode:
  enabled: true

prompt:
  - segments: ["session"]

rprompt:
  - segments: ["vim"]

session:
  type: "session"

vim:
  style: "powerline"
  template: "{{ if .Normal }} NORMAL {{ end }}"
`

	cfg, err := ParseLayoutYAML([]byte(raw))
	require.NoError(t, err)
	require.Contains(t, cfg.Segments, "vim")
	assert.Equal(t, VIM, cfg.Segments["vim"].Type)
}

func TestParseLayoutYAMLAllowsTopLevelMetadataTables(t *testing.T) {
	raw := `
palette:
  bg: "#101010"
  fg: "#f0f0f0"

upgrade:
  source: "github"
  auto: false
  notice: true

maps:
  shell_name:
    zsh: "z"

var:
  app: "prompto"

prompt:
  - segments: ["session"]

session:
  type: "session"
`

	cfg, err := ParseLayoutYAML([]byte(raw))
	require.NoError(t, err)
	require.Len(t, cfg.Prompt, 1)
	require.Contains(t, cfg.Segments, "session")
	assert.Equal(t, "#101010", string(cfg.Palette["bg"]))
	require.NotNil(t, cfg.Upgrade)
	assert.Equal(t, "github", string(cfg.Upgrade.Source))
	require.NotNil(t, cfg.Maps)
	assert.Equal(t, "z", cfg.Maps.GetShellName("zsh"))
	assert.Equal(t, "prompto", cfg.Var["app"])
}

func TestParseLayoutYAMLSupportsSecondaryAndTransient(t *testing.T) {
	raw := `
secondary:
  - segments: ["session"]

transient:
  - segments: ["session"]

prompt:
  - segments: ["session"]

session:
  type: "session"
`

	cfg, err := ParseLayoutYAML([]byte(raw))
	require.NoError(t, err)
	require.Len(t, cfg.SecondaryPrompt, 1)
	require.Len(t, cfg.TransientPrompt, 1)
	assert.Equal(t, []string{"session"}, cfg.SecondaryPrompt[0].Segments)
	assert.Equal(t, []string{"session"}, cfg.TransientPrompt[0].Segments)
}

func TestParseLayoutYAMLRejectsSecondaryPromptTopLevelConfig(t *testing.T) {
	raw := `
secondary_prompt:
  - segments: ["session"]

prompt:
  - segments: ["session"]

session:
  type: "session"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "use secondary")
}

func TestParseLayoutYAMLRejectsTransientPromptTopLevelConfig(t *testing.T) {
	raw := `
transient_prompt:
  - segments: ["session"]

prompt:
  - segments: ["session"]

session:
  type: "session"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "use transient")
}

func TestParseLayoutYAMLRejectsTransientRPromptTopLevelConfig(t *testing.T) {
	raw := `
transient_rprompt:
  - segments: ["session"]

prompt:
  - segments: ["session"]

session:
  type: "session"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "use rtransient")
}

func TestParseLayoutYAMLRejectsTopLevelVimConfig(t *testing.T) {
	raw := `
vim:
  enabled: true

prompt:
  - segments: ["session"]

session:
  type: "session"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "use vim-mode")
}

func TestLayoutConfigApplyMetadata(t *testing.T) {
	layout := &LayoutConfig{
		Palette: color.Palette{
			"bg": "#000000",
		},
		Var: map[string]any{
			"key": "value",
		},
		VimMode: &VimConfig{
			Enabled: true,
		},
		SecondaryPrompt: []PromptLayout{{Segments: []string{"session"}}},
		TransientPrompt: []PromptLayout{{Segments: []string{"session"}}},
	}

	target := &Config{}
	layout.ApplyMetadata(target)

	assert.Equal(t, color.Ansi("#000000"), target.Palette["bg"])
	assert.Equal(t, "value", target.Var["key"])
	require.NotNil(t, target.VimMode)
	assert.True(t, target.VimMode.Enabled)
	require.NotNil(t, target.SecondaryPrompt)
	require.NotNil(t, target.TransientPrompt)
}
