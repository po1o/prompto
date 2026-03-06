package config

import (
	"testing"

	"github.com/jandedobbeleer/oh-my-posh/src/color"
	"github.com/jandedobbeleer/oh-my-posh/src/segments/options"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigMerge(t *testing.T) {
	testCases := []struct {
		baseConfig     *Config
		overrideConfig *Config
		expectedResult *Config
		name           string
		expectError    bool
	}{
		{
			name: "merge basic options",
			baseConfig: &Config{
				FinalSpace:  true,
				Async:       false,
				AccentColor: "red",
			},
			overrideConfig: &Config{
				FinalSpace: false,
				Async:      true,
			},
			expectedResult: &Config{
				FinalSpace:  false,
				Async:       true,
				AccentColor: "red",
				extended:    true,
			},
			expectError: false,
		},
		{
			name: "merge with nil override",
			baseConfig: &Config{
				FinalSpace: true,
			},
			overrideConfig: nil,
			expectedResult: &Config{
				FinalSpace: true,
			},
			expectError: true,
		},
		{
			name: "merge console title template",
			baseConfig: &Config{
				ConsoleTitleTemplate: "Base Title",
			},
			overrideConfig: &Config{
				ConsoleTitleTemplate: "Override Title",
			},
			expectedResult: &Config{
				ConsoleTitleTemplate: "Override Title",
				extended:             true,
			},
			expectError: false,
		},
		{
			name: "merge variables map",
			baseConfig: &Config{
				Var: map[string]any{
					"base_var":   "base_value",
					"shared_var": "base_shared",
				},
			},
			overrideConfig: &Config{
				Var: map[string]any{
					"added_var":  "added_value",
					"shared_var": "override_shared",
				},
			},
			expectedResult: &Config{
				Var: map[string]any{
					"base_var":   "base_value",
					"added_var":  "added_value",
					"shared_var": "override_shared",
				},
				extended: true,
			},
			expectError: false,
		},
		{
			name: "merge blocks with matching alignment",
			baseConfig: &Config{
				Blocks: []*Block{
					{
						Alignment: "left",
						Type:      "prompt",
						Segments: []*Segment{
							{Type: "path", Options: options.Map{"style": "full"}},
						},
					},
				},
			},
			overrideConfig: &Config{
				Blocks: []*Block{
					{
						Alignment: "left",
						Type:      "prompt",
						Segments: []*Segment{
							{Type: "path", Options: options.Map{"style": "short"}},
						},
					},
				},
			},
			expectedResult: &Config{
				Blocks: []*Block{
					{
						Alignment: "left",
						Type:      "prompt",
						Segments: []*Segment{
							{Type: "path", Options: options.Map{"style": "short"}},
						},
					},
				},
				extended: true,
			},
			expectError: false,
		},
		{
			name: "merge blocks with different segment types",
			baseConfig: &Config{
				Blocks: []*Block{
					{
						Alignment: "left",
						Type:      "prompt",
						Segments: []*Segment{
							{Type: "path", Alias: "override", Options: options.Map{"style": "full"}},
						},
					},
				},
			},
			overrideConfig: &Config{
				Blocks: []*Block{
					{
						Alignment: "left",
						Type:      "prompt",
						Segments: []*Segment{
							{Type: "git", Alias: "override", Options: options.Map{"branch_icon": "branch"}},
						},
					},
				},
			},
			expectedResult: &Config{
				Blocks: []*Block{
					{
						Alignment: "left",
						Type:      "prompt",
						Segments: []*Segment{
							{Type: "git", Alias: "override", Options: options.Map{"branch_icon": "branch"}},
						},
					},
				},
				extended: true,
			},
			expectError: false,
		},
		{
			name: "merge segments by index",
			baseConfig: &Config{
				Blocks: []*Block{
					{
						Alignment: "left",
						Type:      "prompt",
						Segments: []*Segment{
							{Type: "path", Options: options.Map{"style": "full"}},
							{Type: "git", Options: options.Map{"branch_icon": ""}},
						},
					},
				},
			},
			overrideConfig: &Config{
				Blocks: []*Block{
					{
						Alignment: "left",
						Type:      "prompt",
						Segments: []*Segment{
							{Type: "path", Index: 1, Options: options.Map{"style": "short"}},
						},
					},
				},
			},
			expectedResult: &Config{
				Blocks: []*Block{
					{
						Alignment: "left",
						Type:      "prompt",
						Segments: []*Segment{
							{Type: "path", Index: 1, Options: options.Map{"style": "short"}},
							{Type: "git", Options: options.Map{"branch_icon": ""}},
						},
					},
				},
				extended: true,
			},
			expectError: false,
		},
		{
			name: "merge block by index",
			baseConfig: &Config{
				Blocks: []*Block{
					{
						Alignment: "left",
						Type:      "prompt",
						Segments: []*Segment{
							{Type: "path", Options: options.Map{"style": "full"}},
							{Type: "git", Options: options.Map{"branch_icon": ""}},
						},
					},
				},
			},
			overrideConfig: &Config{
				Blocks: []*Block{
					{
						Index: 1,
						Segments: []*Segment{
							{Type: "path", Index: 1, Options: options.Map{"style": "short"}},
						},
					},
				},
			},
			expectedResult: &Config{
				Blocks: []*Block{
					{
						Alignment: "left",
						Type:      "prompt",
						Index:     1,
						Segments: []*Segment{
							{Type: "path", Index: 1, Options: options.Map{"style": "short"}},
							{Type: "git", Options: options.Map{"branch_icon": ""}},
						},
					},
				},
				extended: true,
			},
			expectError: false,
		},
		{
			name: "merge palette colors",
			baseConfig: &Config{
				Palette: color.Palette{
					"primary":   "blue",
					"secondary": "green",
				},
			},
			overrideConfig: &Config{
				Palette: color.Palette{
					"primary": "red",
					"accent":  "yellow",
				},
			},
			expectedResult: &Config{
				Palette: color.Palette{
					"primary":   "red",
					"secondary": "green",
					"accent":    "yellow",
				},
				extended: true,
			},
			expectError: false,
		},
		{
			name: "preserve extends field",
			baseConfig: &Config{
				Extends: "/path/to/base.json",
			},
			overrideConfig: &Config{
				Extends: "/path/to/override.json",
			},
			expectedResult: &Config{
				Extends:  "/path/to/base.json",
				extended: true,
			},
			expectError: false,
		},
		{
			name: "merge tooltips slice",
			baseConfig: &Config{
				Tooltips: []*Segment{
					{Type: "git", Tips: []string{"git"}},
				},
			},
			overrideConfig: &Config{
				Tooltips: []*Segment{
					{Type: "path", Tips: []string{"pwd"}},
				},
			},
			expectedResult: &Config{
				Tooltips: []*Segment{
					{Type: "git", Tips: []string{"git"}},
					{Type: "path", Tips: []string{"pwd"}},
				},
				extended: true,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.baseConfig.merge(tc.overrideConfig)

			if tc.expectError {
				require.Error(t, err, tc.name)
				return
			}

			require.NoError(t, err, tc.name)
			assert.EqualExportedValues(t, tc.expectedResult, tc.baseConfig, tc.name)
		})
	}
}

func TestConfigMergeEdgeCases(t *testing.T) {
	testCases := []struct {
		baseConfig     *Config
		overrideConfig *Config
		name           string
		expectError    bool
	}{
		{
			name:           "nil base config",
			baseConfig:     nil,
			overrideConfig: &Config{},
			expectError:    true,
		},
		{
			name:           "empty configs",
			baseConfig:     &Config{},
			overrideConfig: &Config{},
			expectError:    false,
		},
		{
			name: "override with empty blocks",
			baseConfig: &Config{
				Blocks: []*Block{
					{Alignment: "left", Type: "prompt"},
				},
			},
			overrideConfig: &Config{
				Blocks: []*Block{},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.baseConfig.merge(tc.overrideConfig)

			if tc.expectError {
				require.Error(t, err, tc.name)
				return
			}

			require.NoError(t, err, tc.name)
			if tc.baseConfig != nil {
				assert.True(t, tc.baseConfig.extended, tc.name)
			}
		})
	}
}
