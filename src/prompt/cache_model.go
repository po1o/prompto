package prompt

import (
	"sync"
	"time"

	"github.com/jandedobbeleer/oh-my-posh/src/cache"
	"github.com/jandedobbeleer/oh-my-posh/src/color"
	"github.com/jandedobbeleer/oh-my-posh/src/config"
)

type segmentRenderCache struct {
	Text       string
	Foreground color.Ansi
	Background color.Ansi
	RenderedAt time.Time
}

var (
	deviceCacheMu sync.Mutex
	deviceCache   = make(map[string]segmentRenderCache)
)

func (e *Engine) applySegmentCacheBeforeExecute(segment *config.Segment) (reused bool) {
	if e.CompiledConfig == nil {
		return false
	}

	entry, found, cacheKey, strategy, explicit := e.getSegmentCache(segment)
	if !found {
		return false
	}

	if !explicit {
		segment.SetText(entry.Text)
		segment.Foreground = entry.Foreground
		segment.Background = entry.Background
		return false
	}

	duration := segment.Cache.Duration
	if duration.IsEmpty() || duration == cache.INFINITE {
		e.applySegmentCacheEntry(segment, entry)
		e.markSegmentDone(segment)
		e.markSegmentRendered(segment, entry.RenderedAt)
		_ = cacheKey
		_ = strategy
		return true
	}

	expiresIn := time.Duration(duration.Seconds()) * time.Second
	if expiresIn <= 0 {
		e.applySegmentCacheEntry(segment, entry)
		e.markSegmentDone(segment)
		e.markSegmentRendered(segment, entry.RenderedAt)
		return true
	}

	if time.Since(entry.RenderedAt) <= expiresIn {
		e.applySegmentCacheEntry(segment, entry)
		e.markSegmentDone(segment)
		e.markSegmentRendered(segment, entry.RenderedAt)
		return true
	}

	return false
}

func (e *Engine) storeSegmentCache(segment *config.Segment, renderedAt time.Time) {
	if e.CompiledConfig == nil || !segment.Enabled {
		return
	}

	cacheKey, strategy := e.cacheKeyForSegment(segment)
	entry := segmentRenderCache{
		Text:       segment.Text(),
		Foreground: segment.ResolveForeground(),
		Background: segment.ResolveBackground(),
		RenderedAt: renderedAt,
	}

	e.cacheMu.Lock()
	switch strategy {
	case config.Session:
		if e.sessionCache == nil {
			e.sessionCache = make(map[string]segmentRenderCache)
		}
		e.sessionCache[cacheKey] = entry
	case config.Folder:
		if e.folderCache == nil {
			e.folderCache = make(map[string]segmentRenderCache)
		}
		e.folderCache[cacheKey] = entry
	case config.Device:
		deviceCacheMu.Lock()
		deviceCache[cacheKey] = entry
		deviceCacheMu.Unlock()
	}
	e.cacheMu.Unlock()
}

func (e *Engine) getSegmentCache(segment *config.Segment) (segmentRenderCache, bool, string, config.Strategy, bool) {
	cacheKey, strategy := e.cacheKeyForSegment(segment)
	explicit := segment.Cache != nil

	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()

	switch strategy {
	case config.Session:
		entry, ok := e.sessionCache[cacheKey]
		return entry, ok, cacheKey, strategy, explicit
	case config.Folder:
		entry, ok := e.folderCache[cacheKey]
		return entry, ok, cacheKey, strategy, explicit
	case config.Device:
		deviceCacheMu.Lock()
		entry, ok := deviceCache[cacheKey]
		deviceCacheMu.Unlock()
		return entry, ok, cacheKey, strategy, explicit
	default:
		return segmentRenderCache{}, false, cacheKey, strategy, explicit
	}
}

func (e *Engine) cacheKeyForSegment(segment *config.Segment) (string, config.Strategy) {
	if segment.Cache == nil {
		return segment.Name() + "::" + e.Env.Pwd(), config.Folder
	}

	switch segment.Cache.Strategy {
	case config.Session:
		return segment.Name(), config.Session
	case config.Device:
		return segment.Name(), config.Device
	case config.Folder:
		fallthrough
	default:
		return segment.Name() + "::" + e.Env.Pwd(), config.Folder
	}
}

func (e *Engine) applySegmentCacheEntry(segment *config.Segment, entry segmentRenderCache) {
	segment.Enabled = true
	segment.SetText(entry.Text)
	if entry.Foreground != "" {
		segment.Foreground = entry.Foreground
	}
	if entry.Background != "" {
		segment.Background = entry.Background
	}
}

func (e *Engine) executeWithoutLegacySegmentCache(segment *config.Segment) {
	original := segment.Cache
	segment.Cache = nil
	segment.Execute(e.Env)
	segment.Cache = original
}
