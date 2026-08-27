package prompt

import (
	"strings"
	"testing"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/color"
	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/maps"
	"github.com/po1o/prompto/src/runtime"
	"github.com/po1o/prompto/src/shell"
	"github.com/po1o/prompto/src/template"
	"github.com/po1o/prompto/src/terminal"

	"github.com/stretchr/testify/require"
)

func TestPrimaryUsesLayoutOrderAndFiller(t *testing.T) {
	engine := newLayoutTestEngine(t, &config.LayoutConfig{
		Prompt: []config.PromptLayout{
			{Segments: []string{"left_a", "left_b"}, Filler: "."},
			{Segments: []string{"left_c"}},
		},
		RPrompt: []config.PromptLayout{
			{Segments: []string{"right_a"}},
		},
		Segments: map[string]*config.Segment{
			"left_a":  {Type: config.TEXT, Alias: "left_a", Template: "A"},
			"left_b":  {Type: config.TEXT, Alias: "left_b", Template: "B"},
			"right_a": {Type: config.TEXT, Alias: "right_a", Template: "R"},
			"left_c":  {Type: config.TEXT, Alias: "left_c", Template: "C"},
		},
	})

	got := engine.Primary()
	rgot := engine.RPrompt()
	iAB := strings.Index(got, "AB")
	iNewlineC := strings.Index(got, "\nC")

	require.GreaterOrEqual(t, iAB, 0)
	require.GreaterOrEqual(t, iNewlineC, 0)
	require.True(t, iAB < iNewlineC)
	require.Equal(t, "R", rgot)
}

func TestSecondaryUsesLayout(t *testing.T) {
	engine := newLayoutTestEngine(t, &config.LayoutConfig{
		SecondaryPrompt: []config.PromptLayout{
			{Segments: []string{"sec_a"}},
			{Segments: []string{"sec_b"}},
		},
		Segments: map[string]*config.Segment{
			"sec_a": {Type: config.TEXT, Alias: "sec_a", Template: "S1"},
			"sec_b": {Type: config.TEXT, Alias: "sec_b", Template: "S2"},
		},
	})

	got := engine.ExtraPrompt(Secondary)
	require.Equal(t, "S1\nS2", got)
}

func TestTransientUsesLayoutLeftAndRight(t *testing.T) {
	engine := newLayoutTestEngine(t, &config.LayoutConfig{
		TransientPrompt: []config.PromptLayout{
			{Segments: []string{"transient_left"}},
		},
		TransientRPrompt: []config.PromptLayout{
			{Segments: []string{"transient_right"}},
		},
		Segments: map[string]*config.Segment{
			"transient_left":  {Type: config.TEXT, Alias: "transient_left", Template: "TL"},
			"transient_right": {Type: config.TEXT, Alias: "transient_right", Template: "TR"},
		},
	})

	left := engine.ExtraPrompt(Transient)
	right := engine.TransientRPrompt()

	require.Equal(t, "TL", left)
	require.Equal(t, "TR", right)
}

func TestPrimaryInlinesMultilineRightPromptIntoPrimary(t *testing.T) {
	engine := newLayoutTestEngine(t, &config.LayoutConfig{
		Prompt: []config.PromptLayout{
			{Segments: []string{"left_1"}},
			{Segments: []string{"left_2"}},
		},
		RPrompt: []config.PromptLayout{
			{Segments: []string{"right_1"}},
			{Segments: []string{"right_2"}},
		},
		Segments: map[string]*config.Segment{
			"left_1":  {Type: config.TEXT, Alias: "left_1", Template: "L1"},
			"left_2":  {Type: config.TEXT, Alias: "left_2", Template: "L2"},
			"right_1": {Type: config.TEXT, Alias: "right_1", Template: "R1"},
			"right_2": {Type: config.TEXT, Alias: "right_2", Template: "R2"},
		},
	})

	got := engine.Primary()

	require.Contains(t, got, "L1")
	require.Contains(t, got, "R1")
	require.Contains(t, got, "L2")
	require.NotContains(t, got, "R2")
	require.Equal(t, "R2", engine.RPrompt())
}

func TestTransientInlinesMultilineRightPromptIntoTransient(t *testing.T) {
	engine := newLayoutTestEngine(t, &config.LayoutConfig{
		TransientPrompt: []config.PromptLayout{
			{Segments: []string{"left_1"}},
			{Segments: []string{"left_2"}},
		},
		TransientRPrompt: []config.PromptLayout{
			{Segments: []string{"right_1"}},
			{Segments: []string{"right_2"}},
		},
		Segments: map[string]*config.Segment{
			"left_1":  {Type: config.TEXT, Alias: "left_1", Template: "TL1"},
			"left_2":  {Type: config.TEXT, Alias: "left_2", Template: "TL2"},
			"right_1": {Type: config.TEXT, Alias: "right_1", Template: "TR1"},
			"right_2": {Type: config.TEXT, Alias: "right_2", Template: "TR2"},
		},
	})

	got := engine.ExtraPrompt(Transient)

	require.Contains(t, got, "TL1")
	require.Contains(t, got, "TR1")
	require.Contains(t, got, "TL2")
	require.NotContains(t, got, "TR2")
	require.Equal(t, "TR2", engine.TransientRPrompt())
}

func TestPrimaryInlineMultilineRightPromptLeavesLastRowInRPrompt(t *testing.T) {
	flags := &runtime.Flags{
		Shell:         shell.ZSH,
		Eval:          true,
		TerminalWidth: 80,
	}

	env := &runtime.Terminal{}
	env.Init(flags)

	template.Cache = &cache.Template{
		SimpleTemplate: cache.SimpleTemplate{
			Shell: shell.ZSH,
		},
		Segments: maps.NewConcurrent[any](),
	}
	template.Init(env, nil, nil)

	originalPlain := terminal.Plain
	terminal.Init(shell.ZSH)
	terminal.Colors = &color.Defaults{}
	terminal.Plain = false
	t.Cleanup(func() {
		terminal.Plain = originalPlain
	})

	engine := &Engine{
		Env: env,
		Config: &config.Config{
			CursorPadding: true,
		},
		LayoutConfig: &config.LayoutConfig{
			Prompt: []config.PromptLayout{
				{Segments: []string{"left_1"}},
				{Segments: []string{"left_2"}},
			},
			RPrompt: []config.PromptLayout{
				{Segments: []string{"right_1"}},
				{Segments: []string{"right_2"}},
			},
			Segments: map[string]*config.Segment{
				"left_1":  {Type: config.TEXT, Alias: "left_1", Template: "L1"},
				"left_2":  {Type: config.TEXT, Alias: "left_2", Template: "L2"},
				"right_1": {Type: config.TEXT, Alias: "right_1", Template: "R1"},
				"right_2": {Type: config.TEXT, Alias: "right_2", Template: "R2"},
			},
		},
	}

	got := engine.Primary()
	ps1, _, _ := strings.Cut(got, "\nRPROMPT=")

	require.Contains(t, got, "RPROMPT=$'")
	require.NotContains(t, got, "\x1b7")
	require.NotContains(t, got, "\x1b8")
	require.Contains(t, ps1, "R1")
	require.NotContains(t, ps1, "R2")
}

func TestExtraPromptSupportsValidErrorAndDebug(t *testing.T) {
	engine := newLayoutTestEngine(t, &config.LayoutConfig{
		Segments: map[string]*config.Segment{},
	})
	engine.Config.ValidLine = &config.Segment{Template: "VALID"}
	engine.Config.ErrorLine = &config.Segment{Template: "ERROR"}
	engine.Config.DebugPrompt = &config.Segment{Template: "DEBUG"}

	require.Equal(t, "VALID", engine.ExtraPrompt(Valid))
	require.Equal(t, "ERROR", engine.ExtraPrompt(Error))
	require.Equal(t, "DEBUG", engine.ExtraPrompt(Debug))
}

func TestPrimaryMirrorsRightAlignedDiamondSegmentSeparators(t *testing.T) {
	engine := newLayoutTestEngine(t, &config.LayoutConfig{
		RPrompt: []config.PromptLayout{
			{Segments: []string{"right_git"}},
		},
		Segments: map[string]*config.Segment{
			"right_git": {
				Type:          config.TEXT,
				Alias:         "right_git",
				Template:      "R",
				LeadingGlyph:  "",
				TrailingGlyph: "\uE0B0",
			},
		},
	})

	_ = engine.Primary()
	rgot := engine.RPrompt()
	require.Contains(t, rgot, "\uE0B2R")
	require.NotContains(t, rgot, "R\uE0B0")
}

func newLayoutTestEngine(t *testing.T, layout *config.LayoutConfig) *Engine {
	flags := &runtime.Flags{
		Shell:         shell.GENERIC,
		TerminalWidth: 80,
	}

	env := &runtime.Terminal{}
	env.Init(flags)

	template.Cache = &cache.Template{
		SimpleTemplate: cache.SimpleTemplate{
			Shell: shell.GENERIC,
		},
		Segments: maps.NewConcurrent[any](),
	}
	template.Init(env, nil, nil)

	originalPlain := terminal.Plain
	terminal.Init(shell.GENERIC)
	terminal.Colors = &color.Defaults{}
	terminal.Plain = true
	t.Cleanup(func() {
		terminal.Plain = originalPlain
	})

	return &Engine{
		Env:          env,
		Config:       &config.Config{},
		LayoutConfig: layout,
		Plain:        true,
	}
}

// A segment that resolves to no text normally disappears; keep_when_empty
// holds its place so the shapes around it do not reflow.
func TestPrimaryKeepWhenEmptyDrawsGlyphsWithoutText(t *testing.T) {
	newEngine := func(keep bool) *Engine {
		return newLayoutTestEngine(t, &config.LayoutConfig{
			Prompt: []config.PromptLayout{
				{Segments: []string{"before", "empty", "after"}},
			},
			Segments: map[string]*config.Segment{
				"before": {Type: config.TEXT, Alias: "before", Template: "B", Background: "red", Foreground: "white"},
				"empty": {
					Type:          config.TEXT,
					Alias:         "empty",
					Template:      "",
					Background:    "blue",
					Foreground:    "white",
					LeadingGlyph:  "[",
					TrailingGlyph: "]",
					KeepWhenEmpty: keep,
				},
				"after": {Type: config.TEXT, Alias: "after", Template: "A", Background: "green", Foreground: "white"},
			},
		})
	}

	dropped := newEngine(false).Primary()
	require.NotContains(t, dropped, "[")
	require.NotContains(t, dropped, "]")

	kept := newEngine(true).Primary()
	require.Contains(t, kept, "[")
	require.Contains(t, kept, "]")
}

// A line closes itself: where a line and its last segment both define a
// trailing separator, the line's is the one drawn.
func TestBlockTrailingSeparatorWinsOverTheLastSegment(t *testing.T) {
	shared := &config.Segment{
		Type:          config.TEXT,
		Alias:         "shared",
		Template:      "S",
		Background:    "blue",
		Foreground:    "white",
		TrailingGlyph: ")",
	}

	engine := newLayoutTestEngine(t, &config.LayoutConfig{
		Prompt: []config.PromptLayout{
			{Segments: []string{"shared"}, TrailingGlyph: ">"},
		},
		TransientPrompt: []config.PromptLayout{
			{Segments: []string{"shared"}, TrailingGlyph: "]"},
		},
		Segments: map[string]*config.Segment{"shared": shared},
	})

	primary := engine.Primary()
	require.Contains(t, primary, ">")
	require.NotContains(t, primary, ")")

	// The same definition on a second line takes that line's separator, not a
	// leftover from the first.
	transient := engine.ExtraPrompt(Transient)
	require.Contains(t, transient, "]")
	require.NotContains(t, transient, ">")

	require.Equal(t, ")", shared.TrailingGlyph, "the definition itself must be untouched")
}

// Right-aligned lines mirror their separators. The alias table pins that every
// alias has a mirror; this pins that the mirroring is applied to a real render,
// in the right direction. rounded_thin is used deliberately: its mirror was the
// one missing from the hand-written table, and identity would pass unnoticed
// with a symmetric pair.
func TestRPromptMirrorsSegmentSeparators(t *testing.T) {
	engine := newLayoutTestEngine(t, &config.LayoutConfig{
		RPrompt: []config.PromptLayout{
			{Segments: []string{"right"}},
		},
		Segments: map[string]*config.Segment{
			"right": {
				Type:  config.TEXT,
				Alias: "right",
				// Written as it would be for a left-aligned line: closes with
				// the right-facing thin cap, opens flat.
				TrailingGlyph: "\uE0B5",
				Template:      "R",
				Background:    "blue",
				Foreground:    "white",
			},
		},
	})

	_ = engine.Primary()
	got := engine.RPrompt()

	// On the right the block opens instead, so the cap has to face the other way.
	require.Contains(t, got, "\uE0B7", "the trailing cap should mirror into a leading one")
	require.NotContains(t, got, "\uE0B5", "the unmirrored cap points away from its block")
}

// force is a config key, not just the --force flag. Clone used to reset it
// before every render, so a segment that asked to render its whitespace was
// dropped exactly as if it had never set the key.
func TestSegmentForceRendersWhitespaceOnlyText(t *testing.T) {
	definition := &config.Segment{
		Type:     config.TEXT,
		Alias:    "blank",
		Template: "   ",
		Force:    true,
	}

	engine := newLayoutTestEngine(t, &config.LayoutConfig{
		Prompt: []config.PromptLayout{{Segments: []string{"blank", "after"}}},
		Segments: map[string]*config.Segment{
			"blank": definition,
			"after": {Type: config.TEXT, Alias: "after", Template: "B"},
		},
	})

	require.Equal(t, "   B", engine.Primary(), "the forced segment should keep its whitespace")
}

// force overrides the emptiness test and nothing else. A segment held back by
// exclude_folders, a width gate, or `prompto toggle` is not "empty" — it was
// decided against before a template was ever rendered — and force must not
// resurrect it. Folding force into the entry guard instead of the emptiness
// test is what silently breaks all three at once.
func TestSegmentForceDoesNotOverrideTheEnablementGates(t *testing.T) {
	for name, definition := range map[string]*config.Segment{
		"excluded folder": {
			Type:           config.TEXT,
			Alias:          "gated",
			Template:       "G",
			Force:          true,
			ExcludeFolders: []string{".*"},
		},
		// `prompto toggle` is the third gate and behaves the same way, but it
		// reads the session toggle cache rather than the segment, so it is not
		// reachable from this harness.
		"max width": {
			Type:     config.TEXT,
			Alias:    "gated",
			Template: "G",
			Force:    true,
			MaxWidth: 1,
		},
	} {
		t.Run(name, func(t *testing.T) {
			engine := newLayoutTestEngine(t, &config.LayoutConfig{
				Prompt: []config.PromptLayout{{Segments: []string{"gated", "after"}}},
				Segments: map[string]*config.Segment{
					"gated": definition,
					"after": {Type: config.TEXT, Alias: "after", Template: "B"},
				},
			})

			require.Equal(t, "B", engine.Primary(), "the gated segment must stay hidden")
		})
	}
}
