package config

import (
	"testing"
	"time"

	"github.com/jandedobbeleer/oh-my-posh/src/cache"
	"github.com/jandedobbeleer/oh-my-posh/src/cli/upgrade"
	"github.com/jandedobbeleer/oh-my-posh/src/color"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime/mock"
	"github.com/jandedobbeleer/oh-my-posh/src/shell"
	"github.com/jandedobbeleer/oh-my-posh/src/template"

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
func TestUpgradeFeatures(t *testing.T) {
	cases := []struct {
		Case                  string
		ExpectedFeats         shell.Features
		UpgradeCacheKeyExists bool
		AutoUpgrade           bool
		Force                 bool
		DisplayNotice         bool
		AutoUpgradeKey        bool
		NoticeKey             bool
	}{
		{
			Case:                  "cache exists, no force",
			UpgradeCacheKeyExists: true,
			ExpectedFeats:         0,
		},
		{
			Case:          "auto upgrade enabled",
			AutoUpgrade:   true,
			ExpectedFeats: shell.Upgrade,
		},
		{
			Case:           "auto upgrade via cache",
			AutoUpgradeKey: true,
			ExpectedFeats:  shell.Upgrade,
		},
		{
			Case:          "notice enabled, no auto upgrade",
			DisplayNotice: true,
			ExpectedFeats: shell.Notice,
		},
		{
			Case:          "notice via cache, no auto upgrade",
			NoticeKey:     true,
			ExpectedFeats: shell.Notice,
		},
		{
			Case:                  "force upgrade ignores cache",
			UpgradeCacheKeyExists: true,
			Force:                 true,
			AutoUpgrade:           true,
			ExpectedFeats:         shell.Upgrade,
		},
	}

	for _, tc := range cases {
		if tc.UpgradeCacheKeyExists {
			cache.Set(cache.Device, upgrade.CACHEKEY, "", cache.INFINITE)
		}

		if tc.AutoUpgradeKey {
			cache.Set(cache.Device, AUTOUPGRADE, true, cache.INFINITE)
		}

		if tc.NoticeKey {
			cache.Set(cache.Device, UPGRADENOTICE, true, cache.INFINITE)
		}

		cfg := &Config{
			Upgrade: &upgrade.Config{
				Auto:          tc.AutoUpgrade,
				Force:         tc.Force,
				DisplayNotice: tc.DisplayNotice,
			},
		}

		got := cfg.upgradeFeatures()
		assert.Equal(t, tc.ExpectedFeats, got, tc.Case)

		cache.DeleteAll(cache.Device)
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
			expected: shell.Daemon,
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

		cfg := &Config{
			Upgrade: &upgrade.Config{},
		}

		got := cfg.Features(env, tc.daemon)
		assert.Equal(t, tc.expected, got, tc.name)
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
			Upgrade: &upgrade.Config{},
			Vim:     tc.vim,
		}

		got := cfg.Features(env, false)
		assert.Equal(t, tc.expected, got, tc.name)
	}
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
