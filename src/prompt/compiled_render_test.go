package prompt

import (
	"strings"
	"testing"

	"github.com/jandedobbeleer/oh-my-posh/src/cache"
	"github.com/jandedobbeleer/oh-my-posh/src/color"
	"github.com/jandedobbeleer/oh-my-posh/src/config"
	"github.com/jandedobbeleer/oh-my-posh/src/maps"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime"
	"github.com/jandedobbeleer/oh-my-posh/src/shell"
	"github.com/jandedobbeleer/oh-my-posh/src/template"
	"github.com/jandedobbeleer/oh-my-posh/src/terminal"

	"github.com/stretchr/testify/require"
)

func TestPrimaryUsesCompiledLayoutOrderAndFiller(t *testing.T) {
	engine := newCompiledTestEngine(t, &config.CompiledConfig{
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
	iAB := strings.Index(got, "AB")
	iR := strings.Index(got, "R")
	iNewlineC := strings.Index(got, "\nC")

	require.GreaterOrEqual(t, iAB, 0)
	require.GreaterOrEqual(t, iR, 0)
	require.GreaterOrEqual(t, iNewlineC, 0)
	require.True(t, iAB < iR)
	require.True(t, iR < iNewlineC)
	require.Contains(t, got, ".")
}

func TestSecondaryUsesCompiledLayout(t *testing.T) {
	engine := newCompiledTestEngine(t, &config.CompiledConfig{
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

func newCompiledTestEngine(t *testing.T, compiled *config.CompiledConfig) *Engine {
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
		Env:            env,
		Config:         &config.Config{},
		CompiledConfig: compiled,
		Plain:          true,
	}
}
