package config

import (
	"testing"

	"github.com/po1o/prompto/src/color"
	"github.com/po1o/prompto/src/runtime"
	"github.com/po1o/prompto/src/runtime/mock"
	"github.com/po1o/prompto/src/segments"
	"github.com/po1o/prompto/src/segments/options"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v3"
)

type panicCacheKeyWriter struct{}

func (panicCacheKeyWriter) Enabled() bool { return true }

func (panicCacheKeyWriter) Template() string { return "" }

func (panicCacheKeyWriter) SetText(string) {}

func (panicCacheKeyWriter) SetIndex(int) {}

func (panicCacheKeyWriter) Text() string { return "" }

func (panicCacheKeyWriter) Init(options.Provider, runtime.Environment) {}

func (panicCacheKeyWriter) CacheKey() (string, bool) { panic("cache key should not be called") }

const (
	cwd = "Projects/prompto"
)

func TestMapSegmentWriterCanMap(t *testing.T) {
	sc := &Segment{
		Type: SESSION,
	}
	env := new(mock.Environment)
	err := sc.MapSegmentWithWriter(env)
	assert.NoError(t, err)
	assert.NotNil(t, sc.writer)
}

func TestMapSegmentWriterCannotMap(t *testing.T) {
	sc := &Segment{
		Type: "nilwriter",
	}
	env := new(mock.Environment)
	err := sc.MapSegmentWithWriter(env)
	assert.Error(t, err)
}

func TestParseYAMLConfigWithProperties(t *testing.T) {
	segmentYAML := `
type: path
style: powerline
properties:
  style: folder
`
	segment := &Segment{}
	err := yaml.Unmarshal([]byte(segmentYAML), segment)
	assert.NoError(t, err)
	assert.NotNil(t, segment.Options)
	assert.Equal(t, "folder", segment.Options.String("style", ""))
}

func TestParseYAMLConfigWithOptions(t *testing.T) {
	segmentYAML := `
type: path
style: powerline
options:
  style: folder
`
	segment := &Segment{}
	err := yaml.Unmarshal([]byte(segmentYAML), segment)
	assert.NoError(t, err)
	assert.NotNil(t, segment.Options)
	assert.Equal(t, "folder", segment.Options.String("style", ""))
}

func TestShouldIncludeFolder(t *testing.T) {
	cases := []struct {
		Case     string
		Included bool
		Excluded bool
		Expected bool
	}{
		{Case: "Include", Included: true, Excluded: false, Expected: true},
		{Case: "Exclude", Included: false, Excluded: true, Expected: false},
		{Case: "Include & Exclude", Included: true, Excluded: true, Expected: false},
		{Case: "!Include & !Exclude", Included: false, Excluded: false, Expected: false},
	}
	for _, tc := range cases {
		env := new(mock.Environment)
		env.On("GOOS").Return(runtime.LINUX)
		env.On("Home").Return("")
		env.On("Pwd").Return(cwd)
		env.On("DirMatchesOneOf", cwd, []string{"Projects/prompto"}).Return(tc.Included)
		env.On("DirMatchesOneOf", cwd, []string{"Projects/nope"}).Return(tc.Excluded)
		segment := &Segment{
			IncludeFolders: []string{"Projects/prompto"},
			ExcludeFolders: []string{"Projects/nope"},
			env:            env,
		}
		got := segment.shouldIncludeFolder()
		assert.Equal(t, tc.Expected, got, tc.Case)
	}
}

func TestGetColors(t *testing.T) {
	cases := []struct {
		Case       string
		Expected   color.Ansi
		Default    color.Ansi
		Region     string
		Profile    string
		Templates  []string
		Background bool
	}{
		{Case: "No template - foreground", Expected: "color", Background: false, Default: "color"},
		{Case: "No template - background", Expected: "color", Background: true, Default: "color"},
		{Case: "Nil template", Expected: "color", Default: "color", Templates: nil},
		{
			Case:     "Template - default",
			Expected: "color",
			Default:  "color",
			Templates: []string{
				"{{if contains \"john\" .Profile}}color2{{end}}",
			},
			Profile: "doe",
		},
		{
			Case:     "Template - override",
			Expected: "color2",
			Default:  "color",
			Templates: []string{
				"{{if contains \"john\" .Profile}}color2{{end}}",
			},
			Profile: "john",
		},
		{
			Case:     "Template - override multiple",
			Expected: "color3",
			Default:  "color",
			Templates: []string{
				"{{if contains \"doe\" .Profile}}color2{{end}}",
				"{{if contains \"john\" .Profile}}color3{{end}}",
			},
			Profile: "john",
		},
		{
			Case:     "Template - override multiple no match",
			Expected: "color",
			Default:  "color",
			Templates: []string{
				"{{if contains \"doe\" .Profile}}color2{{end}}",
				"{{if contains \"philip\" .Profile}}color3{{end}}",
			},
			Profile: "john",
		},
	}
	for _, tc := range cases {
		segment := &Segment{
			writer: &segments.Aws{
				Profile: tc.Profile,
				Region:  tc.Region,
			},
		}

		if tc.Background {
			segment.Background = tc.Default
			segment.BackgroundTemplates = tc.Templates
			bgColor := segment.ResolveBackground()
			assert.Equal(t, tc.Expected, bgColor, tc.Case)
			continue
		}

		segment.Foreground = tc.Default
		segment.ForegroundTemplates = tc.Templates
		fgColor := segment.ResolveForeground()
		assert.Equal(t, tc.Expected, fgColor, tc.Case)
	}
}

func TestEvaluateNeeds(t *testing.T) {
	cases := []struct {
		Segment *Segment
		Case    string
		Needs   []string
	}{
		{
			Case: "No needs",
			Segment: &Segment{
				Template: "foo",
			},
		},
		{
			Case: "Template needs",
			Segment: &Segment{
				Template: "{{ .Segments.Git.URL }}",
			},
			Needs: []string{"Git"},
		},
		{
			Case: "Template & Foreground needs",
			Segment: &Segment{
				Template:            "{{ .Segments.Git.URL }}",
				ForegroundTemplates: []string{"foo", "{{ .Segments.Os.Icon }}"},
			},
			Needs: []string{"Git", "Os"},
		},
		{
			Case: "Template & Foreground & Background needs",
			Segment: &Segment{
				Template:            "{{ .Segments.Git.URL }}",
				ForegroundTemplates: []string{"foo", "{{ .Segments.Os.Icon }}"},
				BackgroundTemplates: []string{"bar", "{{ .Segments.Exit.Icon }}"},
			},
			Needs: []string{"Git", "Os", "Exit"},
		},
	}
	for _, tc := range cases {
		tc.Segment.evaluateNeeds()
		assert.Equal(t, tc.Needs, tc.Segment.Needs, tc.Case)
	}
}

func TestGetPendingTextDefaults(t *testing.T) {
	segment := &Segment{}
	enabled, text, background := segment.GetPendingText("", nil)

	assert.True(t, enabled)
	assert.Equal(t, "\uf254 ...", text)
	assert.Equal(t, color.Ansi(""), background)
}

func TestGetPendingTextUsesGlobalConfigOverrides(t *testing.T) {
	segment := &Segment{}
	cfg := &Config{
		RenderPendingIcon:       "⌛ ",
		RenderPendingBackground: "red",
	}

	enabled, text, background := segment.GetPendingText("cached", cfg)

	assert.True(t, enabled)
	assert.Equal(t, "⌛ cached", text)
	assert.Equal(t, color.Ansi("red"), background)
}

func TestGetPendingTextUsesSegmentOverrides(t *testing.T) {
	segment := &Segment{
		RenderPendingIcon:       "⏱ ",
		RenderPendingBackground: "blue",
	}
	cfg := &Config{
		RenderPendingIcon:       "⌛ ",
		RenderPendingBackground: "red",
	}

	enabled, text, background := segment.GetPendingText("cached", cfg)

	assert.True(t, enabled)
	assert.Equal(t, "⏱ cached", text)
	assert.Equal(t, color.Ansi("blue"), background)
}

func TestDaemonCacheKeySkipsWriterCacheKeyWhenNoExplicitCache(t *testing.T) {
	env := new(mock.Environment)
	env.On("Pwd").Return("/tmp")

	segment := &Segment{
		Alias:  "git",
		writer: panicCacheKeyWriter{},
		env:    env,
	}

	assert.Equal(t, "daemon_cache_git_/tmp", segment.DaemonCacheKey())
}
