package config

import (
	"fmt"
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
	assert.Equal(t, "\uE0B2", cfg.Prompt[0].LeadingGlyph)
	assert.Equal(t, "\uE0B0", cfg.Prompt[0].TrailingGlyph)
	assert.Empty(t, cfg.Prompt[0].LeadingStyle)
	assert.Empty(t, cfg.Prompt[0].TrailingStyle)
	assert.Empty(t, cfg.Prompt[0].LeadingSeparator)
	assert.Empty(t, cfg.Prompt[0].TrailingSeparator)

	require.Len(t, cfg.RPrompt, 1)
	assert.Equal(t, []string{"git.main"}, cfg.RPrompt[0].Segments)
	assert.Equal(t, "\uE0B2", cfg.RPrompt[0].LeadingGlyph)
	assert.Equal(t, "\uE0B0", cfg.RPrompt[0].TrailingGlyph)
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
	assert.Equal(t, "\uE0B6", cfg.Segments["path"].LeadingGlyph)
	assert.Equal(t, ">", cfg.Segments["path"].TrailingGlyph)
	assert.Equal(t, GIT, cfg.Segments["git.main"].Type)
	assert.Equal(t, "git.main", cfg.Segments["git.main"].Alias)
	assert.Equal(t, 20, cfg.Segments["git.main"].Options["branch_max_length"])
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
	assert.Equal(t, "", cfg.Prompt[0].LeadingGlyph)
	assert.Equal(t, "\uE0B0", cfg.Prompt[0].TrailingGlyph)

	require.Len(t, cfg.RPrompt, 1)
	assert.Equal(t, "\uE0B2", cfg.RPrompt[0].LeadingGlyph)
	assert.Equal(t, "", cfg.RPrompt[0].TrailingGlyph)
}

func TestParseLayoutYAMLStyleShortcutOnPromptLinesRoundedThin(t *testing.T) {
	raw := `
prompt:
  - segments: ["session"]
    style: "rounded_thin"

rprompt:
  - segments: ["session"]
    style: "rounded_thin"

session:
  type: "session"
`

	cfg, err := ParseLayoutYAML([]byte(raw))
	require.NoError(t, err)

	require.Len(t, cfg.Prompt, 1)
	assert.Equal(t, "", cfg.Prompt[0].LeadingGlyph)
	assert.Equal(t, "\uE0B5", cfg.Prompt[0].TrailingGlyph)

	require.Len(t, cfg.RPrompt, 1)
	assert.Equal(t, "\uE0B7", cfg.RPrompt[0].LeadingGlyph)
	assert.Equal(t, "", cfg.RPrompt[0].TrailingGlyph)
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
	assert.Equal(t, "", segment.LeadingGlyph)
	assert.Equal(t, "\uE0B0", segment.TrailingGlyph)
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

func TestParseLayoutYAMLReturnsErrorForUnsupportedTimeFormat(t *testing.T) {
	raw := `
prompt:
  - segments: ["time"]

time:
  type: "time"
  options:
    time_format: "RFC3339"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "unsupported time_format")
	assert.ErrorContains(t, err, "RFC3339")
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

func TestParseLayoutYAMLReturnsErrorForUnknownTopLevelScalarKey(t *testing.T) {
	raw := `
promt:
  - segments: ["session"]

session:
  type: "session"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "unknown top-level key")
	assert.ErrorContains(t, err, "promt")
}

func TestParseLayoutYAMLReturnsErrorForUnknownTopLevelNestedTable(t *testing.T) {
	raw := `
custom:
  main:
    style: "powerline"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "unknown top-level key")
	assert.ErrorContains(t, err, "custom")
}

func TestParseLayoutYAMLDefaultsCursorPaddingToTrue(t *testing.T) {
	raw := `
prompt:
  - segments: ["session"]

session:
  type: "session"
`

	cfg, err := ParseLayoutYAML([]byte(raw))
	require.NoError(t, err)
	assert.True(t, cfg.CursorPadding)
}

func TestParseLayoutYAMLUsesCursorPaddingKey(t *testing.T) {
	raw := `
cursor_padding: false

prompt:
  - segments: ["session"]

session:
  type: "session"
`

	cfg, err := ParseLayoutYAML([]byte(raw))
	require.NoError(t, err)
	assert.False(t, cfg.CursorPadding)
}

func TestParseLayoutYAMLAllowsKnownNestedSegmentTables(t *testing.T) {
	raw := `
prompt:
  - segments: ["git.main"]

git:
  main:
    style: "powerline"
`

	cfg, err := ParseLayoutYAML([]byte(raw))
	require.NoError(t, err)
	require.Len(t, cfg.Segments, 1)
	assert.Equal(t, GIT, cfg.Segments["git.main"].Type)
	assert.Equal(t, "git.main", cfg.Segments["git.main"].Alias)
}

func TestParseLayoutYAMLReturnsErrorForDirectPromptGlyphs(t *testing.T) {
	raw := `
prompt:
  - segments: ["session"]
    leading_glyph: "<"

session:
  type: "session"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "does not allow leading_glyph")
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

func TestParseLayoutYAMLReturnsErrorForDirectSegmentGlyphs(t *testing.T) {
	raw := `
prompt:
  - segments: ["session"]

session:
  type: "session"
  leading_glyph: "<"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "does not allow leading_glyph")
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
	require.NotNil(t, cfg.Maps)
	assert.Equal(t, "z", cfg.Maps.GetShellName("zsh"))
	assert.Equal(t, "prompto", cfg.Var["app"])
}

func TestParseLayoutYAMLRejectsRemovedUpgradeMetadataTable(t *testing.T) {
	raw := `
upgrade:
  source: "github"

prompt:
  - segments: ["session"]

session:
  type: "session"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, `top-level key "upgrade" no longer exists`)
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

func TestParseLayoutYAMLSupportsValidErrorAndDebugPrompts(t *testing.T) {
	raw := `
valid_line:
  template: "ok"
error_line:
  template: "err"
debug_prompt:
  template: "dbg"

prompt:
  - segments: ["session"]

session:
  type: "session"
`

	cfg, err := ParseLayoutYAML([]byte(raw))
	require.NoError(t, err)
	require.NotNil(t, cfg.ValidLine)
	require.NotNil(t, cfg.ErrorLine)
	require.NotNil(t, cfg.DebugPrompt)
	assert.Equal(t, TEXT, cfg.ValidLine.Type)
	assert.Equal(t, TEXT, cfg.ErrorLine.Type)
	assert.Equal(t, TEXT, cfg.DebugPrompt.Type)
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
		ValidLine:       &Segment{Template: "ok"},
		ErrorLine:       &Segment{Template: "err"},
		DebugPrompt:     &Segment{Template: "dbg"},
		SecondaryPrompt: []PromptLayout{{Segments: []string{"session"}}},
		TransientPrompt: []PromptLayout{{Segments: []string{"session"}}},
	}

	target := &Config{}
	layout.ApplyMetadata(target)

	assert.Equal(t, color.Ansi("#000000"), target.Palette["bg"])
	assert.Equal(t, "value", target.Var["key"])
	require.NotNil(t, target.VimMode)
	assert.True(t, target.VimMode.Enabled)
	require.NotNil(t, target.ValidLine)
	require.NotNil(t, target.ErrorLine)
	require.NotNil(t, target.DebugPrompt)
	assert.True(t, target.HasSecondary)
	assert.True(t, target.HasTransient)
}

// The render styles are gone: a config carrying one has to say so instead of
// silently rendering as something else.
func TestParseLayoutYAMLReturnsErrorForRemovedSegmentRenderStyles(t *testing.T) {
	for _, style := range []string{"plain", "diamond", "accordion"} {
		t.Run(style, func(t *testing.T) {
			raw := `
prompt:
  - segments: ["session"]

session:
  type: "session"
  style: "` + style + `"
`

			_, err := ParseLayoutYAML([]byte(raw))
			require.Error(t, err)
			assert.ErrorContains(t, err, "is not a separator alias")
		})
	}
}

func TestParseLayoutYAMLReturnsErrorForRemovedLineRenderStyles(t *testing.T) {
	for _, style := range []string{"plain", "diamond", "accordion"} {
		t.Run(style, func(t *testing.T) {
			raw := `
prompt:
  - style: "` + style + `"
    segments: ["session"]

session:
  type: "session"
`

			_, err := ParseLayoutYAML([]byte(raw))
			require.Error(t, err)
			assert.ErrorContains(t, err, "is not a separator alias")
		})
	}
}

// style is a config-level shortcut; the engine must never see it back.
func TestParseLayoutYAMLDropsStyleAfterCompiling(t *testing.T) {
	raw := `
prompt:
  - segments: ["session"]

session:
  type: "session"
  style: "rounded"
`

	cfg, err := ParseLayoutYAML([]byte(raw))
	require.NoError(t, err)

	segment := cfg.Segments["session"]
	require.NotNil(t, segment)
	assert.Equal(t, "", segment.LeadingGlyph)
	assert.Equal(t, "\uE0B4", segment.TrailingGlyph)
}

func TestParseLayoutYAMLReadsKeepWhenEmpty(t *testing.T) {
	raw := `
prompt:
  - segments: ["session"]

session:
  type: "session"
  keep_when_empty: true
`

	cfg, err := ParseLayoutYAML([]byte(raw))
	require.NoError(t, err)
	assert.True(t, cfg.Segments["session"].KeepWhenEmpty)
}

// The keys this format used to carry must fail loudly: silently ignoring them
// renders something the user did not ask for and gives no clue why.
func TestParseLayoutYAMLReturnsErrorForRemovedSegmentKeys(t *testing.T) {
	// The full message is asserted, not just the key: one key is a substring of
	// another, and the replacement is the part a user actually needs.
	for key, expected := range map[string]struct{ value, message string }{
		"powerline_symbol":         {`">"`, "session uses powerline_symbol, use trailing_separator instead"},
		"leading_powerline_symbol": {`"<"`, "session uses leading_powerline_symbol, use leading_separator instead"},
		"invert_powerline":         {"true", "session uses invert_powerline, which no longer exists"},
		"leading_diamond":          {`"("`, "session uses leading_diamond, use leading_separator or leading_style instead"},
		"trailing_diamond":         {`")"`, "session uses trailing_diamond, use trailing_separator or trailing_style instead"},
	} {
		t.Run(key, func(t *testing.T) {
			raw := `
prompt:
  - segments: ["session"]

session:
  type: "session"
  ` + key + `: ` + expected.value + `
`

			_, err := ParseLayoutYAML([]byte(raw))
			require.Error(t, err)
			assert.EqualError(t, err, expected.message)
		})
	}
}

func TestParseLayoutYAMLReturnsErrorForRemovedLineKeys(t *testing.T) {
	raw := `
prompt:
  - leading_diamond: "("
    segments: ["session"]

session:
  type: "session"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "leading_diamond")
}

// keep_when_empty is about one segment's text, so a line cannot claim it.
func TestParseLayoutYAMLReturnsErrorForKeepWhenEmptyOnLine(t *testing.T) {
	raw := `
prompt:
  - keep_when_empty: true
    segments: ["session"]

session:
  type: "session"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "keep_when_empty")
}

// Tooltips and the debug/valid/error lines are segments too, so the separator
// vocabulary has to reach them and the engine fields must stay out.
func TestParseLayoutYAMLCompilesSeparatorsOnTooltips(t *testing.T) {
	raw := `
prompt:
  - segments: ["session"]

session:
  type: "session"

tooltips:
  - type: "aws"
    tips: ["aws"]
    style: "rounded"
`

	cfg, err := ParseLayoutYAML([]byte(raw))
	require.NoError(t, err)
	require.Len(t, cfg.Tooltips, 1)

	// A tooltip only ever renders in a right-aligned block, and nothing mirrors
	// it on the way there, so style has to compile to the right-aligned
	// orientation here: the alias opens the tooltip rather than closing it.
	assert.Equal(t, "\uE0B6", cfg.Tooltips[0].LeadingGlyph)
	assert.Empty(t, cfg.Tooltips[0].TrailingGlyph)
}

// Every separator key has to orient the same way on a tooltip, not just the
// style shortcut: a user copying keys from a working rprompt segment into a
// tooltip should get the same shape. Compared against a named segment mirrored
// the way layoutBlock mirrors one onto a right-aligned line.
func TestTooltipSeparatorsMatchARightAlignedSegment(t *testing.T) {
	for _, key := range []string{
		`style: "rounded"`,
		`leading_style: "rounded"`,
		`trailing_style: "rounded"`,
		`leading_separator: "Y"`,
		`trailing_separator: "X"`,
	} {
		t.Run(key, func(t *testing.T) {
			onLine, err := ParseLayoutYAML(fmt.Appendf(nil,
				"rprompt:\n  - segments: [\"s\"]\ns:\n  type: \"session\"\n  %s\n", key))
			require.NoError(t, err)

			expected := onLine.Segments["s"].Clone()
			expected.MirrorSeparators(false)

			asTooltip, err := ParseLayoutYAML(fmt.Appendf(nil,
				"prompt:\n  - segments: [\"s\"]\ns:\n  type: \"session\"\ntooltips:\n  - type: \"aws\"\n    tips: [\"aws\"]\n    %s\n", key))
			require.NoError(t, err)
			require.Len(t, asTooltip.Tooltips, 1)

			assert.Equal(t, expected.LeadingGlyph, asTooltip.Tooltips[0].LeadingGlyph, "leading")
			assert.Equal(t, expected.TrailingGlyph, asTooltip.Tooltips[0].TrailingGlyph, "trailing")
		})
	}
}

// A typeless tooltip is rejected rather than silently never rendering, which is
// what it did before tooltips went through the segment decoder.
func TestParseLayoutYAMLRejectsATooltipWithoutAType(t *testing.T) {
	raw := `
prompt:
  - segments: ["session"]

session:
  type: "session"

tooltips:
  - tips: ["aws"]
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "tooltip 0 is missing type")
}

func TestParseLayoutYAMLRejectsGlyphsOnTooltips(t *testing.T) {
	raw := `
prompt:
  - segments: ["session"]

session:
  type: "session"

tooltips:
  - type: "aws"
    tips: ["aws"]
    leading_glyph: "<"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "does not allow leading_glyph")
}

func TestParseLayoutYAMLCompilesSeparatorsOnExtraLines(t *testing.T) {
	raw := `
prompt:
  - segments: ["session"]

session:
  type: "session"

valid_line:
  template: "V"
  leading_separator: "("

error_line:
  template: "E"
  style: "rounded"
`

	cfg, err := ParseLayoutYAML([]byte(raw))
	require.NoError(t, err)

	require.NotNil(t, cfg.ValidLine)
	assert.Equal(t, "(", cfg.ValidLine.LeadingGlyph)
	assert.Equal(t, TEXT, cfg.ValidLine.Type)

	require.NotNil(t, cfg.ErrorLine)
	assert.Equal(t, "\uE0B4", cfg.ErrorLine.TrailingGlyph)

	// They are addressed by role, never by name, so they must not become
	// segments a prompt line could reference.
	assert.NotContains(t, cfg.Segments, "valid_line")
	assert.NotContains(t, cfg.Segments, "error_line")
}

func TestParseLayoutYAMLRejectsGlyphsOnExtraLines(t *testing.T) {
	raw := `
prompt:
  - segments: ["session"]

session:
  type: "session"

valid_line:
  template: "V"
  trailing_glyph: ">"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "does not allow trailing_glyph")
}

// debug_prompt behaved differently depending on whether it carried a type.
func TestParseLayoutYAMLRejectsRemovedStyleOnDebugPromptWithoutType(t *testing.T) {
	raw := `
prompt:
  - segments: ["session"]

session:
  type: "session"

debug_prompt:
  template: "D"
  style: "diamond"
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "is not a separator alias")
}

// The mirror table is derived from the alias table; this pins that they stay in
// step, which a hand-written mirror did not.
func TestMirrorGlyphCoversEverySeparatorAlias(t *testing.T) {
	for alias, pair := range separatorAliases {
		assert.Equal(t, pair.right, MirrorGlyph(pair.left, false), "%s left", alias)
		assert.Equal(t, pair.left, MirrorGlyph(pair.right, false), "%s right", alias)
	}
}

// A glyph the user wrote themselves has no mirror to look up.
func TestMirrorGlyphLeavesUnknownGlyphsAlone(t *testing.T) {
	assert.Equal(t, ">", MirrorGlyph(">", false))
	assert.Equal(t, "", MirrorGlyph("", false))
}

// options and cache are the only segment keys that decode to a map, so a
// segment configured with nothing else looked exactly like a group of named
// instances and was parsed as one — leaving the segment itself unregistered and
// reporting the line that referenced it as the error.
func TestParseLayoutYAMLSegmentWithOnlyMapValuedKeys(t *testing.T) {
	for name, raw := range map[string]string{
		"options": "prompt:\n  - segments: [\"git\"]\ngit:\n  options:\n    fetch_status: true\n",
		"cache":   "prompt:\n  - segments: [\"git\"]\ngit:\n  cache:\n    duration: \"1h\"\n",
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := ParseLayoutYAML([]byte(raw))
			require.NoError(t, err)
			require.Contains(t, cfg.Segments, "git")
			assert.Equal(t, GIT, cfg.Segments["git"].Type)
		})
	}
}

// The nested form still has to work, or the fix above would have traded one
// misparse for the other.
func TestParseLayoutYAMLStillReadsNestedInstances(t *testing.T) {
	raw := `
prompt:
  - segments: ["git.work"]

git:
  work:
    template: " a "
  personal:
    template: " b "
`

	cfg, err := ParseLayoutYAML([]byte(raw))
	require.NoError(t, err)
	require.Contains(t, cfg.Segments, "git.work")
	require.Contains(t, cfg.Segments, "git.personal")
	assert.Equal(t, GIT, cfg.Segments["git.work"].Type)
}

// A misspelled key on an otherwise bare segment falls out of the segment branch
// and is read as a group of named instances. The error has to name the key, or
// it describes a nested table the user never wrote.
func TestParseLayoutYAMLNamesAMisspelledSegmentKey(t *testing.T) {
	raw := `
prompt:
  - segments: ["git"]

git:
  templat: " x "
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "templat")
	assert.ErrorContains(t, err, "is not a segment key")
}

// An instance named after a segment key makes the table read as one segment,
// silently losing every other instance in it. Reported against the table that
// caused it rather than the prompt line that referenced the missing instance.
func TestParseLayoutYAMLRejectsAnInstanceNameCollidingWithASegmentKey(t *testing.T) {
	raw := `
prompt:
  - segments: ["git.work"]

git:
  cache:
    template: " a "
  work:
    template: " b "
`

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
	assert.ErrorContains(t, err, "named instances")
	assert.ErrorContains(t, err, "work")
}

// A console config compiles separator aliases to ASCII. The graphical glyphs
// are unreadable on a text console, and the alias is the same one either way,
// so the theme does not have to be rewritten to move between them.
func TestParseLayoutYAMLCompilesConsoleSeparators(t *testing.T) {
	body := `
prompt:
  - segments: ["path"]

path:
  type: "text"
  style: "powerline"
  template: " ~ "
`

	graphical, err := ParseLayoutYAML([]byte(body))
	require.NoError(t, err)
	assert.Equal(t, "\ue0b0", graphical.Segments["path"].TrailingGlyph)

	console, err := ParseLayoutYAMLFrom([]byte(body), "theme.console.yaml")
	require.NoError(t, err)
	assert.Equal(t, ">", console.Segments["path"].TrailingGlyph)
}

// The translation runs on the compiled glyph, so a config that writes the Nerd
// Font character out by hand is converted too — it is exactly as unreadable on
// a console as the alias that produces it.
func TestParseLayoutYAMLConvertsLiteralGlyphsOnConsole(t *testing.T) {
	raw := `

prompt:
  - segments: ["path"]
    trailing_separator: "\ue0b4"

path:
  type: "text"
  trailing_separator: "\ue0b0"
  template: " ~ "
`

	cfg, err := ParseLayoutYAMLFrom([]byte(raw), "theme.console.yaml")
	require.NoError(t, err)
	assert.Equal(t, ">", cfg.Segments["path"].TrailingGlyph)
	assert.Equal(t, ")", cfg.Prompt[0].TrailingGlyph)
}

// flame, pixel and lego have no ASCII that reads as the same shape. Dropping
// the glyph leaves the segment flat, which is what a console font would show
// for it anyway; substituting an unrelated character would not.
func TestParseLayoutYAMLDropsSeparatorsWithNoConsoleShape(t *testing.T) {
	raw := `

prompt:
  - segments: ["path"]

path:
  type: "text"
  style: "flame"
  template: " ~ "
`

	cfg, err := ParseLayoutYAMLFrom([]byte(raw), "theme.console.yaml")
	require.NoError(t, err)
	assert.Empty(t, cfg.Segments["path"].TrailingGlyph)
}

// Mirroring has to know the charset. ">" is an ordinary character a graphical
// theme may use as a literal separator, and flipping it there would turn a
// glyph the theme chose deliberately.
func TestMirrorGlyphOnlyMirrorsASCIIOnConsole(t *testing.T) {
	assert.Equal(t, ">", MirrorGlyph(">", false), "graphical themes keep a literal >")
	assert.Equal(t, "<", MirrorGlyph(">", true))
	assert.Equal(t, "(", MirrorGlyph(")", true))
	assert.Equal(t, "\ue0b0", MirrorGlyph("\ue0b2", false))
}

// Every graphical alias needs a console counterpart or a deliberate omission,
// so adding an alias cannot silently leave the console rendering it as nothing.
func TestConsoleSeparatorAliasesCoverTheGraphicalOnes(t *testing.T) {
	withoutConsoleShape := map[string]bool{"flame": true, "pixel": true, "lego": true}

	for alias := range separatorAliases {
		_, hasConsole := consoleSeparatorAliases[alias]
		require.Equal(t, !withoutConsoleShape[alias], hasConsole,
			"%s: console table and the documented omissions disagree", alias)
	}

	for alias := range consoleSeparatorAliases {
		require.Contains(t, separatorAliases, alias, "%s is console-only", alias)
	}
}

// Console mode is derived from the file name, not declared inside the file. A
// console config is already named config.console.yaml — that is how Resolve
// finds it — so the same document means different things read from the two
// paths, and neither has to say so.
func TestParseLayoutYAMLDerivesConsoleFromTheFileName(t *testing.T) {
	body := `
prompt:
  - segments: ["path"]

path:
  type: "text"
  style: "powerline"
  template: " ~ "
`

	for name, want := range map[string]string{
		"config.yaml":               "",
		"config.yml":                "",
		"config.console.yaml":       ">",
		"~/themes/mine.console.yml": ">",
		"":                          "",
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := ParseLayoutYAMLFrom([]byte(body), name)
			require.NoError(t, err)
			assert.Equal(t, want, cfg.Segments["path"].TrailingGlyph)
		})
	}
}
