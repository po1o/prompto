package prompt

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

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

type testDeviceCache struct {
	entries map[string]DeviceCacheEntry
	lastTTL time.Duration
}

func (store *testDeviceCache) Get(key string) (DeviceCacheEntry, bool) {
	value, ok := store.entries[key]
	return value, ok
}

func (store *testDeviceCache) Set(key string, value DeviceCacheEntry, ttl time.Duration) {
	store.lastTTL = ttl
	store.entries[key] = value
}

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
	engine.resetSharedProviders()
	_, _ = engine.writeBlockSegments(block)
	engine.resetSharedProviders()
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
	engine.resetSharedProviders()
	_, _ = engine.writeBlockSegments(block)

	key := segment.Name()
	entry := engine.sessionCache[key]
	entry.RenderedAt = time.Now().Add(-48 * time.Hour)
	engine.sessionCache[key] = entry

	engine.resetSharedProviders()
	_, _ = engine.writeBlockSegments(block)
	require.Equal(t, int32(2), atomic.LoadInt32(&count))
}

func TestExplicitSessionCacheExpiredUsesStaleValueForPendingRender(t *testing.T) {
	var count int32
	engine := newCacheTestEngine(&count)
	segment := &config.Segment{
		Type:     config.TEXT,
		Alias:    "cached.pending",
		Template: "A",
		Cache: &config.Cache{
			Strategy: config.Session,
			Duration: cache.ONEDAY,
		},
	}

	key := segment.Name()
	block := &config.Block{Segments: []*config.Segment{segment}}
	_, _ = engine.writeBlockSegments(block)

	engine.sessionCache[key] = segmentRenderCache{
		Text:       "stale",
		Foreground: color.Ansi("red"),
		Background: color.Ansi("blue"),
		RenderedAt: time.Now().Add(-48 * time.Hour),
	}

	reused := engine.applySegmentCacheBeforeExecute(segment)

	require.False(t, reused)
	require.Equal(t, "stale", segment.Text())
	require.Equal(t, color.Ansi("red"), segment.Foreground)
	require.Equal(t, color.Ansi("blue"), segment.Background)
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
	engine.resetSharedProviders()
	_, _ = engine.writeBlockSegments(block)
	engine.resetSharedProviders()
	_, _ = engine.writeBlockSegments(block)

	require.Equal(t, int32(2), atomic.LoadInt32(&count))
}

func TestImplicitGitCacheKeyUsesRepoRootAcrossSubdirectories(t *testing.T) {
	repoRoot := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(repoRoot, "a", "b"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(repoRoot, "c", "d"), 0o755))

	firstPwd := filepath.Join(repoRoot, "a", "b")
	secondPwd := filepath.Join(repoRoot, "c", "d")

	firstEngine := newCacheTestEngineWithPwd(t, firstPwd)
	secondEngine := newCacheTestEngineWithPwd(t, secondPwd)

	firstSegment := &config.Segment{Type: config.GIT, Alias: "git"}
	secondSegment := &config.Segment{Type: config.GIT, Alias: "git"}
	require.NoError(t, firstSegment.MapSegmentWithWriter(firstEngine.Env))
	require.NoError(t, secondSegment.MapSegmentWithWriter(secondEngine.Env))

	firstKey, firstStrategy := firstEngine.cacheKeyForSegment(firstSegment)
	secondKey, secondStrategy := secondEngine.cacheKeyForSegment(secondSegment)

	require.Equal(t, config.Folder, firstStrategy)
	require.Equal(t, config.Folder, secondStrategy)
	require.Equal(t, firstKey, secondKey)
	require.Contains(t, firstKey, repoRoot)
}

func TestStoreSegmentCacheUsesInjectedDeviceCacheForFolderStrategy(t *testing.T) {
	var count int32
	engine := newCacheTestEngine(&count)
	cacheStore := &testDeviceCache{
		entries: map[string]DeviceCacheEntry{},
	}
	engine.SetDeviceCache(cacheStore)

	segment := &config.Segment{
		Type:     config.TEXT,
		Alias:    "text.main",
		Template: "A",
	}
	require.NoError(t, segment.MapSegmentWithWriter(engine.Env))
	segment.Enabled = true
	segment.SetText("cached")

	renderedAt := time.Now()
	engine.storeSegmentCache(segment, renderedAt)

	require.Empty(t, engine.folderCache)
	require.NotEmpty(t, cacheStore.entries)
	require.Equal(t, time.Duration(0), cacheStore.lastTTL)
}

func TestStoreSegmentCacheUsesExplicitDurationForInjectedDeviceCache(t *testing.T) {
	var count int32
	engine := newCacheTestEngine(&count)
	cacheStore := &testDeviceCache{
		entries: map[string]DeviceCacheEntry{},
	}
	engine.SetDeviceCache(cacheStore)

	segment := &config.Segment{
		Type:     config.TEXT,
		Alias:    "text.main",
		Template: "A",
		Cache: &config.Cache{
			Strategy: config.Folder,
			Duration: cache.ONEDAY,
		},
	}
	require.NoError(t, segment.MapSegmentWithWriter(engine.Env))
	segment.Enabled = true
	segment.SetText("cached")

	engine.storeSegmentCache(segment, time.Now())

	require.Equal(t, 24*time.Hour, cacheStore.lastTTL)
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
		Env:          env,
		Config:       &config.Config{},
		LayoutConfig: &config.LayoutConfig{},
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

func newCacheTestEngineWithPwd(t *testing.T, pwd string) *Engine {
	t.Helper()

	flags := &runtime.Flags{
		Shell:         shell.GENERIC,
		TerminalWidth: 80,
		PWD:           pwd,
	}

	env := &runtime.Terminal{}
	env.Init(flags)

	return &Engine{
		Env:           env,
		Config:        &config.Config{},
		LayoutConfig:  &config.LayoutConfig{},
		segmentStates: make(map[string]*segmentAsyncState),
		sessionCache:  make(map[string]segmentRenderCache),
		folderCache:   make(map[string]segmentRenderCache),
	}
}
