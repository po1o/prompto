package segments

import (
	"testing"

	"github.com/jandedobbeleer/oh-my-posh/src/runtime"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime/mock"
	"github.com/jandedobbeleer/oh-my-posh/src/segments/options"

	"github.com/stretchr/testify/assert"
)

func TestVimSegment(t *testing.T) {
	cases := []struct {
		Case        string
		VimMode     string
		ExpMode     string
		ExpEnabled  bool
		ExpNormal   bool
		ExpInsert   bool
		ExpVisual   bool
		ExpReplace  bool
		ExpCommand  bool
		ExpOperator bool
	}{
		{
			Case:       "Empty mode - disabled",
			VimMode:    "",
			ExpEnabled: false,
		},
		{
			Case:       "Normal mode",
			VimMode:    "normal",
			ExpEnabled: true,
			ExpMode:    "normal",
			ExpNormal:  true,
		},
		{
			Case:       "Insert mode",
			VimMode:    "insert",
			ExpEnabled: true,
			ExpMode:    "insert",
			ExpInsert:  true,
		},
		{
			Case:       "Visual mode",
			VimMode:    "visual",
			ExpEnabled: true,
			ExpMode:    "visual",
			ExpVisual:  true,
		},
		{
			Case:       "Replace mode",
			VimMode:    "replace",
			ExpEnabled: true,
			ExpMode:    "replace",
			ExpReplace: true,
		},
		{
			Case:       "Command mode",
			VimMode:    "command",
			ExpEnabled: true,
			ExpMode:    "command",
			ExpCommand: true,
		},
		{
			Case:        "Operator mode",
			VimMode:     "operator",
			ExpEnabled:  true,
			ExpMode:     "operator",
			ExpOperator: true,
		},
	}

	for _, tc := range cases {
		env := new(mock.Environment)
		env.On("Flags").Return(&runtime.Flags{VimMode: tc.VimMode})

		v := &Vim{}
		v.Init(options.Map{}, env)

		enabled := v.Enabled()
		assert.Equal(t, tc.ExpEnabled, enabled, tc.Case)

		if tc.ExpEnabled {
			assert.Equal(t, tc.ExpMode, v.Mode, tc.Case+" - Mode")
			assert.Equal(t, tc.ExpNormal, v.Normal, tc.Case+" - Normal")
			assert.Equal(t, tc.ExpInsert, v.Insert, tc.Case+" - Insert")
			assert.Equal(t, tc.ExpVisual, v.Visual, tc.Case+" - Visual")
			assert.Equal(t, tc.ExpReplace, v.Replace, tc.Case+" - Replace")
			assert.Equal(t, tc.ExpCommand, v.Command, tc.Case+" - Command")
			assert.Equal(t, tc.ExpOperator, v.Operator, tc.Case+" - Operator")
		}
	}
}

func TestVimTemplate(t *testing.T) {
	v := &Vim{}
	assert.Equal(t, " {{ .Mode }} ", v.Template())
}

func TestVimConditionalTemplate(t *testing.T) {
	cases := []struct {
		Case     string
		VimMode  string
		Template string
		Expected string
	}{
		{
			Case:     "Normal mode with conditional",
			VimMode:  "normal",
			Template: "{{ if .Normal }}N{{ else }}I{{ end }}",
			Expected: "N",
		},
		{
			Case:     "Insert mode with conditional",
			VimMode:  "insert",
			Template: "{{ if .Normal }}N{{ else if .Insert }}I{{ else }}?{{ end }}",
			Expected: "I",
		},
		{
			Case:     "Visual mode indicator",
			VimMode:  "visual",
			Template: "{{ if .Visual }}VISUAL{{ else }}{{ .Mode }}{{ end }}",
			Expected: "VISUAL",
		},
	}

	for _, tc := range cases {
		env := new(mock.Environment)
		env.On("Flags").Return(&runtime.Flags{VimMode: tc.VimMode})

		v := &Vim{}
		v.Init(options.Map{}, env)
		v.Enabled()

		got := renderTemplate(env, tc.Template, v)
		assert.Equal(t, tc.Expected, got, tc.Case)
	}
}
