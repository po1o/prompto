package prompt

import (
	"os"
	"testing"

	"github.com/jandedobbeleer/oh-my-posh/src/runtime"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createVimConfig(t *testing.T, template string) string {
	t.Helper()
	content := `{
		"version": 4,
		"blocks": [
			{
				"type": "prompt",
				"alignment": "left",
				"segments": [
					{
						"type": "vim",
						"style": "plain",
						"template": "` + template + `"
					}
				]
			}
		]
	}`

	tmpFile, err := os.CreateTemp("", "omp-vim-*.json")
	require.NoError(t, err)
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	err = tmpFile.Close()
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })
	return tmpFile.Name()
}

func TestVimSegmentCLIRender(t *testing.T) {
	cases := []struct {
		Case       string
		VimMode    string
		Template   string
		Contains   string
		ShouldShow bool
	}{
		{
			Case:       "Normal mode renders",
			VimMode:    "normal",
			Template:   "[{{ .Mode }}]",
			Contains:   "[normal]",
			ShouldShow: true,
		},
		{
			Case:       "Insert mode renders",
			VimMode:    "insert",
			Template:   "[{{ .Mode }}]",
			Contains:   "[insert]",
			ShouldShow: true,
		},
		{
			Case:       "Visual mode with boolean",
			VimMode:    "visual",
			Template:   "{{ if .Visual }}VIS{{ end }}",
			Contains:   "VIS",
			ShouldShow: true,
		},
		{
			Case:       "Normal mode with boolean conditional",
			VimMode:    "normal",
			Template:   "{{ if .Normal }}N{{ else if .Insert }}I{{ end }}",
			Contains:   "N",
			ShouldShow: true,
		},
		{
			Case:       "Empty vim mode - segment hidden",
			VimMode:    "",
			Template:   "[{{ .Mode }}]",
			Contains:   "",
			ShouldShow: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Case, func(t *testing.T) {
			configPath := createVimConfig(t, tc.Template)

			flags := &runtime.Flags{
				ConfigPath: configPath,
				PWD:        "/tmp",
				Shell:      "bash",
				Type:       "primary",
				IsPrimary:  true,
				VimMode:    tc.VimMode,
				Plain:      true,
			}

			eng := New(flags)
			result := eng.Primary()

			if tc.ShouldShow {
				assert.Contains(t, result, tc.Contains, tc.Case)
			} else {
				// When vim mode is empty, segment should not render
				assert.NotContains(t, result, "[", tc.Case)
			}
		})
	}
}
