package prompt

import (
	"sync/atomic"
	"testing"
	"time"

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

type countingProvider struct {
	count *int32
}

func (provider *countingProvider) Execute(e *Engine, source *config.Segment) (sharedExecutionResult, error) {
	atomic.AddInt32(provider.count, 1)
	source.Execute(e.Env)
	return sharedExecutionResult{
		Text:    source.Text(),
		Enabled: source.Enabled,
	}, nil
}

func TestExplicitSessionCacheReusesFreshRender(t *testing.T) {
	var count int32
	engine := newCacheTestEngine(&count)
	segment := &config.Segment{
		Type:     config.TEXT,
		Alias:    "cached.text",
		Template: "A",
		Cache: &config.Cache{
			Strategy: config.Session,
			Duration: cache.ONEDAY,
		},
	}

	block := &config.Block{Segments: []*config.Segment{segment}}
	_, _ = engine.writeBlockSegments(block)
	_, _ = engine.writeBlockSegments(block)

	require.Equal(t, int32(1), atomic.LoadInt32(&count))
}

func TestExplicitSessionCacheExpiredRecomputes(t *testing.T) {
	var count int32
	engine := newCacheTestEngine(&count)
	segment := &config.Segment{
		Type:     config.TEXT,
		Alias:    "cached.expired",
		Template: "A",
		Cache: &config.Cache{
			Strategy: config.Session,
			Duration: cache.ONEDAY,
		},
	}

	block := &config.Block{Segments: []*config.Segment{segment}}
	_, _ = engine.writeBlockSegments(block)

	key := segment.Name()
	entry := engine.sessionCache[key]
	entry.RenderedAt = time.Now().Add(-48 * time.Hour)
	engine.sessionCache[key] = entry

	_, _ = engine.writeBlockSegments(block)
	require.Equal(t, int32(2), atomic.LoadInt32(&count))
}

func TestImplicitCacheAlwaysRecomputes(t *testing.T) {
	var count int32
	engine := newCacheTestEngine(&count)
	segment := &config.Segment{
		Type:     config.TEXT,
		Alias:    "implicit.text",
		Template: "A",
	}

	block := &config.Block{Segments: []*config.Segment{segment}}
	_, _ = engine.writeBlockSegments(block)
	_, _ = engine.writeBlockSegments(block)

	require.Equal(t, int32(2), atomic.LoadInt32(&count))
}

func newCacheTestEngine(count *int32) *Engine {
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

	return &Engine{
		Env:            env,
		Config:         &config.Config{},
		CompiledConfig: &config.CompiledConfig{},
		sharedProviderFactory: map[config.SegmentType]sharedProviderFactory{
			config.TEXT: func() sharedSegmentProvider {
				return &countingProvider{count: count}
			},
		},
		segmentStates: make(map[string]*segmentAsyncState),
		sessionCache:  make(map[string]segmentRenderCache),
		folderCache:   make(map[string]segmentRenderCache),
	}
}
