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
				Type:            config.TEXT,
				Alias:           "right_git",
				Style:           config.Diamond,
				Template:        "R",
				LeadingDiamond:  "",
				TrailingDiamond: "\uE0B0",
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
