package segments

import (
	"testing"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/runtime/mock"
	"github.com/po1o/prompto/src/segments/options"
	"github.com/po1o/prompto/src/template"

	"github.com/stretchr/testify/assert"
)

func TestTextSegment(t *testing.T) {
	cases := []struct {
		Case             string
		ExpectedString   string
		Template         string
		ExpectedDisabled bool
	}{
		{Case: "standard text", ExpectedString: "hello", Template: "hello"},
		{Case: "template text with env var", ExpectedString: "hello world", Template: "{{ .Env.HELLO }} world"},
		{Case: "template text with shell name", ExpectedString: "hello world from terminal", Template: "{{ .Env.HELLO }} world from {{ .Shell }}"},
		{Case: "template text with folder", ExpectedString: "hello world in prompto", Template: "{{ .Env.HELLO }} world in {{ .Folder }}"},
		{Case: "template text with user", ExpectedString: "hello Prompto", Template: "{{ .Env.HELLO }} {{ .UserName }}"},
		{Case: "empty text", Template: "", ExpectedDisabled: true},
		{Case: "empty template result", Template: "{{ .Env.WORLD }}", ExpectedDisabled: true},
	}

	for _, tc := range cases {
		env := new(mock.Environment)
		env.On("PathSeparator").Return("/")
		env.On("Getenv", "HELLO").Return("hello")
		env.On("Getenv", "WORLD").Return("")

		txt := &Text{}
		txt.Init(options.Map{}, env)

		template.Cache = &cache.Template{
			SimpleTemplate: cache.SimpleTemplate{
				UserName: "Prompto",
				HostName: "MyHost",
				Shell:    "terminal",
				Root:     true,
				Folder:   "prompto",
			},
		}

		assert.Equal(t, tc.ExpectedString, renderTemplate(env, tc.Template, txt), tc.Case)
	}
}
