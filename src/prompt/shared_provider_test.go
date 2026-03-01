package prompt

import (
	"sync/atomic"
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
