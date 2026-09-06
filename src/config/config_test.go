package config

import (
	"testing"
	"time"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/color"
	"github.com/po1o/prompto/src/runtime/mock"
	"github.com/po1o/prompto/src/shell"
	"github.com/po1o/prompto/src/template"

	"github.com/stretchr/testify/assert"
)

func TestGetPalette(t *testing.T) {
	palette := color.Palette{
		"red":  "#ff0000",
		"blue": "#0000ff",
	}

	cases := []struct {
		Palettes        *color.Palettes
		Palette         color.Palette
		ExpectedPalette color.Palette
		Case            string
	}{
		{
			Case: "match",
			Palettes: &color.Palettes{
				Template: "{{ .Shell }}",
				List: map[string]color.Palette{
					"bash": palette,
					"zsh": {
						"red":  "#ff0001",
						"blue": "#0000fb",
					},
				},
			},
			ExpectedPalette: palette,
		},
		{
			Case: "no match, no fallback",
			Palettes: &color.Palettes{
				Template: "{{ .Shell }}",
				List: map[string]color.Palette{
					"fish": palette,
					"zsh": {
						"red":  "#ff0001",
						"blue": "#0000fb",
					},
				},
			},
			ExpectedPalette: nil,
		},
		{
			Case: "no match, default",
			Palettes: &color.Palettes{
				Template: "{{ .Shell }}",
				List: map[string]color.Palette{
					"zsh": {
						"red":  "#ff0001",
						"blue": "#0000fb",
					},
				},
			},
			Palette:         palette,
			ExpectedPalette: palette,
		},
		{
			Case:            "no palettes",
			ExpectedPalette: nil,
		},
		{
			Case: "match, with override",
			Palettes: &color.Palettes{
				Template: "{{ .Shell }}",
				List: map[string]color.Palette{
					"bash": {
						"red":    "#ff0001",
						"yellow": "#ffff00",
					},
				},
			},
			Palette: palette,
			ExpectedPalette: color.Palette{
				"red":    "#ff0001",
				"blue":   "#0000ff",
				"yellow": "#ffff00",
			},
		},
	}

	for _, tc := range cases {
		env := &mock.Environment{}
		env.On("Shell").Return("bash")

		template.Cache = &cache.Template{
			SimpleTemplate: cache.SimpleTemplate{
				Shell: "bash",
			},
		}
		template.Init(env, nil, nil)

		cfg := &Config{
			Palette:  tc.Palette,
			Palettes: tc.Palettes,
		}

		got := cfg.getPalette()
		assert.Equal(t, tc.ExpectedPalette, got, tc.Case)
	}
}

func TestFeaturesDaemon(t *testing.T) {
	cases := []struct {
		name     string
		shell    string
		daemon   bool
		expected shell.Features
	}{
		{
			name:     "daemon enabled for zsh",
			shell:    shell.ZSH,
			daemon:   true,
			expected: shell.Daemon | shell.Transient,
		},
		{
			name:   "daemon disabled by flag",
			shell:  shell.ZSH,
			daemon: false,
		},
		{
			name:   "daemon unsupported shell",
			shell:  shell.ELVISH,
			daemon: true,
		},
	}

	for _, tc := range cases {
		env := &mock.Environment{}
		env.On("Shell").Return(tc.shell)

		cfg := &Config{}

		got := cfg.Features(env, tc.daemon)
		assert.Equal(t, tc.expected, got, tc.name)
	}
}

func TestFeaturesDaemonAlwaysEnablesTransientForSupportedShells(t *testing.T) {
	supportedShells := []string{shell.BASH, shell.ZSH, shell.FISH, shell.PWSH}

	for _, sh := range supportedShells {
		env := &mock.Environment{}
		env.On("Shell").Return(sh)

		cfg := &Config{}
		got := cfg.Features(env, true)

		assert.True(t, got&shell.Transient != 0, sh)
	}
}

func TestFeaturesVim(t *testing.T) {
	tests := []struct {
		vim      *VimConfig
		name     string
		expected shell.Features
	}{
		{
			name: "vim enabled",
			vim: &VimConfig{
				Enabled: true,
			},
			expected: shell.VimMode,
		},
		{
			name: "cursor shape implies vim mode",
			vim: &VimConfig{
				CursorShape: true,
			},
			expected: shell.VimMode | shell.VimCursorShape,
		},
		{
			name: "cursor blink implies shape and mode",
			vim: &VimConfig{
				CursorBlink: true,
			},
			expected: shell.VimMode | shell.VimCursorShape | shell.VimCursorBlink,
		},
		{
			name: "shape and blink",
			vim: &VimConfig{
				CursorShape: true,
				CursorBlink: true,
			},
			expected: shell.VimMode | shell.VimCursorShape | shell.VimCursorBlink,
		},
	}

	for _, tc := range tests {
		env := &mock.Environment{}
		env.On("Shell").Return(shell.ZSH)

		cfg := &Config{
			VimMode: tc.vim,
		}

		got := cfg.Features(env, false)
		assert.Equal(t, tc.expected, got, tc.name)
	}
}

func TestFeaturesVimModeAlias(t *testing.T) {
	env := &mock.Environment{}
	env.On("Shell").Return(shell.ZSH)

	cfg := &Config{
		VimMode: &VimConfig{
			Enabled:     true,
			CursorShape: true,
		},
	}

	got := cfg.Features(env, false)
	assert.Equal(t, shell.VimMode|shell.VimCursorShape, got)
}

func TestFeaturesVimModeUsesOnlyVimModeKey(t *testing.T) {
	env := &mock.Environment{}
	env.On("Shell").Return(shell.ZSH)

	cfg := &Config{
		VimMode: &VimConfig{
			Enabled: true,
		},
	}

	got := cfg.Features(env, false)
	assert.Equal(t, shell.VimMode, got)
}

func TestGetDaemonIdleTimeout(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected time.Duration
	}{
		{
			name:     "default when unset",
			value:    "",
			expected: 5 * time.Minute,
		},
		{
			name:     "disabled when none",
			value:    "none",
			expected: 0,
		},
		{
			name:     "valid minutes",
			value:    "12",
			expected: 12 * time.Minute,
		},
		{
			name:     "invalid value falls back to default",
			value:    "invalid",
			expected: 5 * time.Minute,
		},
		{
			name:     "negative value falls back to default",
			value:    "-1",
			expected: 5 * time.Minute,
		},
	}

	for _, tc := range tests {
		cfg := &Config{
			DaemonIdleTimeout: tc.value,
		}

		got := cfg.GetDaemonIdleTimeout()
		assert.Equal(t, tc.expected, got, tc.name)
	}
}

func TestGetDaemonTimeout(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		expected time.Duration
	}{
		{
			name:     "default when unset",
			value:    0,
			expected: 100 * time.Millisecond,
		},
		{
			name:     "default when negative",
			value:    -1,
			expected: 100 * time.Millisecond,
		},
		{
			name:     "valid milliseconds",
			value:    250,
			expected: 250 * time.Millisecond,
		},
	}

	for _, tc := range tests {
		cfg := &Config{
			DaemonTimeout: tc.value,
		}

		got := cfg.GetDaemonTimeout()
		assert.Equal(t, tc.expected, got, tc.name)
	}

	var nilCfg *Config
	assert.Equal(t, 100*time.Millisecond, nilCfg.GetDaemonTimeout())
}

// TestGetRenderTimeout pins the mapping the render marker rests on, including
// the negative case, which reads like "disable" and deliberately is not: the
// caller's stream deadline applies regardless, so suppressing the marker
// entirely would only trade something the user can read for a prompt that keeps
// its placeholders with nothing to explain why.
func TestGetRenderTimeout(t *testing.T) {
	cases := []struct {
		Case       string
		Configured int
		Expected   time.Duration
	}{
		{Case: "unset takes the default", Configured: 0, Expected: 60 * time.Second},
		{Case: "seconds are honoured", Configured: 5, Expected: 5 * time.Second},
		{Case: "a large value is honoured", Configured: 600, Expected: 600 * time.Second},
		{Case: "negative means no deadline of its own", Configured: -1, Expected: 0},
		{Case: "any negative, not just -1", Configured: -99, Expected: 0},
	}

	for _, tc := range cases {
		cfg := &Config{RenderTimeout: tc.Configured}
		assert.Equal(t, tc.Expected, cfg.GetRenderTimeout(), tc.Case)
	}

	var nilConfig *Config
	assert.Equal(t, 60*time.Second, nilConfig.GetRenderTimeout(), "a nil config takes the default")
}

// cache.Get returns the stored map itself, so toggleSegments has to publish a
// fresh copy rather than write into what it read. The daemon holds references
// to previously published maps and reads them from other goroutines — seeding
// sessions, diffing on reload — while renders in other shells call Load, so
// mutating a published map in place is a data race that brings the daemon down
// with "concurrent map read and map write".
func TestToggleSegmentsPublishesACopyInsteadOfMutating(t *testing.T) {
	cache.Delete(cache.Session, cache.TOGGLECACHE)
	t.Cleanup(func() {
		cache.Delete(cache.Session, cache.TOGGLECACHE)
	})

	first := &Config{Layout: &LayoutConfig{Segments: map[string]*Segment{
		"left": {Alias: "left", Toggled: true},
	}}}
	first.toggleSegments()

	published, OK := cache.Get[map[string]bool](cache.Session, cache.TOGGLECACHE)
	assert.True(t, OK)
	assert.True(t, published["left"])

	second := &Config{Layout: &LayoutConfig{Segments: map[string]*Segment{
		"right": {Alias: "right", Toggled: true},
	}}}
	second.toggleSegments()

	assert.False(t, published["right"], "a published toggle map must never be written to again")

	latest, _ := cache.Get[map[string]bool](cache.Session, cache.TOGGLECACHE)
	assert.True(t, latest["left"], "the new map must carry the toggles the old one had")
	assert.True(t, latest["right"])
}
