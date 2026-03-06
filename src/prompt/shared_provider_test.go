package prompt

import (
	"sync/atomic"
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

type countingTextProvider struct {
	count *int32
}

func (provider *countingTextProvider) Execute(e *Engine, source *config.Segment) (sharedExecutionResult, error) {
	atomic.AddInt32(provider.count, 1)
	source.Execute(e.Env)
	return sharedExecutionResult{
		Text:    source.Text(),
		Enabled: source.Enabled,
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
