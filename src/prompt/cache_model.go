package prompt

import (
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/color"
	"github.com/po1o/prompto/src/config"
)

type segmentRenderCache struct {
	RenderedAt time.Time
	Text       string
	Foreground color.Ansi
	Background color.Ansi
}

type DeviceCacheEntry = segmentRenderCache

type DeviceCache interface {
	Get(key string) (DeviceCacheEntry, bool)
	Set(key string, value DeviceCacheEntry, ttl time.Duration)
}

var (
	deviceCacheMu sync.Mutex
	deviceCache   = make(map[string]segmentRenderCache)
)

func (e *Engine) applySegmentCacheBeforeExecute(segment *config.Segment) (reused bool) {
	if e.LayoutConfig == nil {
		return false
	}

	entry, found, explicit := e.getSegmentCache(segment)
	if !found {
		return false
	}

	if !explicit {
		if !e.ensureSegmentWriter(segment) {
			return false
		}

		segment.SetText(entry.Text)
		segment.Foreground = entry.Foreground
		segment.Background = entry.Background
		return false
	}

	duration := segment.Cache.Duration
	if duration.IsEmpty() || duration == cache.INFINITE {
		e.applySegmentCacheEntry(segment, entry)
		return true
	}

	expiresIn := time.Duration(duration.Seconds()) * time.Second
	if expiresIn <= 0 {
		e.applySegmentCacheEntry(segment, entry)
		return true
	}

	if time.Since(entry.RenderedAt) <= expiresIn {
		e.applySegmentCacheEntry(segment, entry)
		return true
	}

	// Keep stale cache visible while recomputing so pending renders have content.
	e.applySegmentCacheEntry(segment, entry)

	return false
}

func (e *Engine) storeSegmentCache(segment *config.Segment, renderedAt time.Time) {
	if e.LayoutConfig == nil || !segment.Enabled {
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
		if e.deviceCache != nil {
			e.deviceCache.Set(cacheKey, entry, segmentCacheTTL(segment))
			break
		}

		if e.folderCache == nil {
			e.folderCache = make(map[string]segmentRenderCache)
		}
		e.folderCache[cacheKey] = entry
	case config.Device:
		if e.deviceCache != nil {
			e.deviceCache.Set(cacheKey, entry, segmentCacheTTL(segment))
			break
		}

		deviceCacheMu.Lock()
		deviceCache[cacheKey] = entry
		deviceCacheMu.Unlock()
	}
	e.cacheMu.Unlock()
}

func (e *Engine) getSegmentCache(segment *config.Segment) (segmentRenderCache, bool, bool) {
	cacheKey, strategy := e.cacheKeyForSegment(segment)
	explicit := segment.Cache != nil

	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()

	switch strategy {
	case config.Session:
		entry, ok := e.sessionCache[cacheKey]
		return entry, ok, explicit
	case config.Folder:
		if e.deviceCache != nil {
			entry, ok := e.deviceCache.Get(cacheKey)
			return entry, ok, explicit
		}

		entry, ok := e.folderCache[cacheKey]
		return entry, ok, explicit
	case config.Device:
		if e.deviceCache != nil {
			entry, ok := e.deviceCache.Get(cacheKey)
			return entry, ok, explicit
		}

		deviceCacheMu.Lock()
		entry, ok := deviceCache[cacheKey]
		deviceCacheMu.Unlock()
		return entry, ok, explicit
	default:
		return segmentRenderCache{}, false, explicit
	}
}

func (e *Engine) cacheKeyForSegment(segment *config.Segment) (string, config.Strategy) {
	if segment.Cache == nil {
		return segment.DaemonCacheKey(), config.Folder
	}

	switch segment.Cache.Strategy {
	case config.Session:
		return segment.Name(), config.Session
	case config.Device:
		return segment.Name(), config.Device
	case config.Folder:
		fallthrough
	default:
		return segment.Name() + "::" + segment.FolderKey(), config.Folder
	}
}

func (e *Engine) applySegmentCacheEntry(segment *config.Segment, entry segmentRenderCache) {
	if !e.ensureSegmentWriter(segment) {
		return
	}

	segment.Enabled = true
	segment.SetText(entry.Text)
	if entry.Foreground != "" {
		segment.Foreground = entry.Foreground
	}
	if entry.Background != "" {
		segment.Background = entry.Background
	}
}

func (e *Engine) ensureSegmentWriter(segment *config.Segment) bool {
	err := segment.MapSegmentWithWriter(e.Env)
	return err == nil
}

// SessionCacheItem describes one rendered segment cached on a session's engine.
type SessionCacheItem struct {
	RenderedAt time.Time
	Key        string
	Text       string
}

// SessionCacheItems returns a snapshot of every segment currently cached for
// this session, sorted by key.
func (e *Engine) SessionCacheItems() []SessionCacheItem {
	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()

	items := make([]SessionCacheItem, 0, len(e.sessionCache))
	for key, entry := range e.sessionCache {
		items = append(items, SessionCacheItem{
			Key:        key,
			Text:       entry.Text,
			RenderedAt: entry.RenderedAt,
		})
	}

	slices.SortFunc(items, func(a, b SessionCacheItem) int {
		return strings.Compare(a.Key, b.Key)
	})

	return items
}

func (e *Engine) SetDeviceCache(cacheStore DeviceCache) {
	e.deviceCache = cacheStore
}

func segmentCacheTTL(segment *config.Segment) time.Duration {
	if segment == nil || segment.Cache == nil || segment.Cache.Duration.IsEmpty() {
		return 0
	}

	return time.Duration(segment.Cache.Duration.Seconds()) * time.Second
}
