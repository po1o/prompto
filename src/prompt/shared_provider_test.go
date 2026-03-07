package prompt

import (
	"sync/atomic"
	"testing"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/color"
	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/maps"
	"github.com/po1o/prompto/src/runtime"
	"github.com/po1o/prompto/src/segments/options"
	"github.com/po1o/prompto/src/shell"
	"github.com/po1o/prompto/src/template"
	"github.com/po1o/prompto/src/terminal"

	"github.com/stretchr/testify/require"
)

type countingTextProvider struct {
	count *int32
}

var sharedStateExecCount int32

type sharedStateWriter struct {
	TextValue string
	Value     int
}

func (provider *countingTextProvider) Execute(e *Engine, source *config.Segment) (sharedExecutionResult, error) {
	atomic.AddInt32(provider.count, 1)
	source.Execute(e.Env)
	return sharedExecutionResult{
		Source: source,
	}, nil
}

func TestSharedProviderExecutesTextSegmentOncePerBlock(t *testing.T) {
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

	terminal.Init(shell.GENERIC)
	terminal.Colors = &color.Defaults{}

	var executionCount int32
	engine := &Engine{
		Env: env,
		sharedProviderFactory: map[config.SegmentType]sharedProviderFactory{
			config.TEXT: func() sharedSegmentProvider {
				return &countingTextProvider{count: &executionCount}
			},
		},
	}

	block := &config.Block{
		Segments: []*config.Segment{
			{Type: config.TEXT, Template: "A", Alias: "one"},
			{Type: config.TEXT, Template: "B", Alias: "two"},
			{Type: config.TEXT, Template: "C", Alias: "three"},
		},
	}

	prompt, length := engine.writeBlockSegments(block)
	require.Equal(t, int32(1), atomic.LoadInt32(&executionCount))
	require.Equal(t, 3, length)
	require.Contains(t, prompt, "A")
	require.Contains(t, prompt, "B")
	require.Contains(t, prompt, "C")
}

func TestSharedProviderExecutesOnceAcrossBlocksWithinRender(t *testing.T) {
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

	terminal.Init(shell.GENERIC)
	terminal.Colors = &color.Defaults{}

	var executionCount int32
	engine := &Engine{
		Env: env,
		sharedProviderFactory: map[config.SegmentType]sharedProviderFactory{
			config.TEXT: func() sharedSegmentProvider {
				return &countingTextProvider{count: &executionCount}
			},
		},
	}

	firstBlock := &config.Block{Segments: []*config.Segment{{Type: config.TEXT, Template: "A", Alias: "first"}}}
	secondBlock := &config.Block{Segments: []*config.Segment{{Type: config.TEXT, Template: "B", Alias: "second"}}}

	engine.resetSharedProviders()
	_, _ = engine.writeBlockSegments(firstBlock)
	_, _ = engine.writeBlockSegments(secondBlock)

	require.Equal(t, int32(1), atomic.LoadInt32(&executionCount))
}

func TestSharedProviderResetsBetweenRenders(t *testing.T) {
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

	terminal.Init(shell.GENERIC)
	terminal.Colors = &color.Defaults{}

	var executionCount int32
	engine := &Engine{
		Env: env,
		sharedProviderFactory: map[config.SegmentType]sharedProviderFactory{
			config.TEXT: func() sharedSegmentProvider {
				return &countingTextProvider{count: &executionCount}
			},
		},
	}

	block := &config.Block{Segments: []*config.Segment{{Type: config.TEXT, Template: "A", Alias: "text"}}}

	engine.resetSharedProviders()
	_, _ = engine.writeBlockSegments(block)
	engine.resetSharedProviders()
	_, _ = engine.writeBlockSegments(block)

	require.Equal(t, int32(2), atomic.LoadInt32(&executionCount))
}

func TestSharedProviderKeepsPerInstanceTemplateWithSharedTypeState(t *testing.T) {
	segmentType := config.SegmentType("shared_state_test")
	previousFactory, hadPreviousFactory := config.Segments[segmentType]
	config.Segments[segmentType] = func() config.SegmentWriter { return &sharedStateWriter{} }
	t.Cleanup(func() {
		if hadPreviousFactory {
			config.Segments[segmentType] = previousFactory
			return
		}

		delete(config.Segments, segmentType)
	})

	atomic.StoreInt32(&sharedStateExecCount, 0)

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

	terminal.Init(shell.GENERIC)
	terminal.Colors = &color.Defaults{}

	engine := &Engine{
		Env: env,
		sharedProviderFactory: map[config.SegmentType]sharedProviderFactory{
			segmentType: func() sharedSegmentProvider {
				return &stateSharedProvider{}
			},
		},
	}

	block := &config.Block{
		Segments: []*config.Segment{
			{Type: segmentType, Alias: "one", Template: "A{{ .Value }}"},
			{Type: segmentType, Alias: "two", Template: "B{{ .Value }}"},
		},
	}

	promptText, _ := engine.writeBlockSegments(block)
	require.Equal(t, int32(1), atomic.LoadInt32(&sharedStateExecCount))
	require.Contains(t, promptText, "A1")
	require.Contains(t, promptText, "B1")
}

func (writer *sharedStateWriter) Enabled() bool {
	count := atomic.AddInt32(&sharedStateExecCount, 1)
	writer.Value = int(count)
	return true
}

func (writer *sharedStateWriter) Template() string {
	return "{{ .Value }}"
}

func (writer *sharedStateWriter) SetText(text string) {
	writer.TextValue = text
}

func (writer *sharedStateWriter) SetIndex(_ int) {}

func (writer *sharedStateWriter) Text() string {
	return writer.TextValue
}

func (writer *sharedStateWriter) Init(_ options.Provider, _ runtime.Environment) {}

func (writer *sharedStateWriter) CacheKey() (string, bool) {
	return "", false
}
