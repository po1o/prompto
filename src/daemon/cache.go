package daemon

import (
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/po1o/prompto/src/prompt"
)

const DefaultDeviceCacheTTL = 7 * 24 * time.Hour

// SegmentRenderValue is the daemon's view of a cached rendered segment.
// It is a type alias for prompt.DeviceCacheEntry, which lets *DeviceCache
// satisfy the prompt.DeviceCache interface directly — no bridge type needed.
type SegmentRenderValue = prompt.DeviceCacheEntry

type deviceCacheEntry struct {
	expiresAt time.Time
	value     SegmentRenderValue
	infinite  bool
}

type DeviceCache struct {
	entries    map[string]deviceCacheEntry
	defaultTTL time.Duration
	mu         sync.RWMutex
}

func NewDeviceCache() *DeviceCache {
	return &DeviceCache{
		entries:    make(map[string]deviceCacheEntry),
		defaultTTL: DefaultDeviceCacheTTL,
	}
}

func (cache *DeviceCache) SetDefaultTTL(ttl time.Duration) {
	cache.mu.Lock()
	cache.defaultTTL = ttl
	cache.mu.Unlock()
}

func (cache *DeviceCache) GetDefaultTTL() time.Duration {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.defaultTTL
}

func (cache *DeviceCache) Set(key string, value SegmentRenderValue, ttl time.Duration) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	effectiveTTL := ttl
	if effectiveTTL == 0 {
		effectiveTTL = cache.defaultTTL
	}

	infinite := effectiveTTL < 0

	cache.entries[key] = deviceCacheEntry{
		value:     value,
		expiresAt: time.Now().Add(effectiveTTL),
		infinite:  infinite,
	}

	if infinite {
		cache.entries[key] = deviceCacheEntry{
			value:    value,
			infinite: true,
		}
	}
}

func (cache *DeviceCache) Get(key string) (SegmentRenderValue, bool) {
	cache.mu.RLock()
	entry, ok := cache.entries[key]
	cache.mu.RUnlock()
	if !ok {
		return SegmentRenderValue{}, false
	}

	if !entry.infinite && time.Now().After(entry.expiresAt) {
		cache.mu.Lock()
		delete(cache.entries, key)
		cache.mu.Unlock()
		return SegmentRenderValue{}, false
	}

	return entry.value, true
}

func (cache *DeviceCache) Delete(key string) {
	cache.mu.Lock()
	delete(cache.entries, key)
	cache.mu.Unlock()
}

func (cache *DeviceCache) Clear() {
	cache.mu.Lock()
	cache.entries = make(map[string]deviceCacheEntry)
	cache.mu.Unlock()
}

func (cache *DeviceCache) Count() int {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return len(cache.entries)
}

// Item describes one cached rendered segment for display.
type Item struct {
	RenderedAt time.Time
	ExpiresAt  time.Time
	Key        string
	Text       string
	Expired    bool
	Forever    bool
}

// Items lists the cache's contents, sorted by key. Expired entries are
// reported rather than skipped: seeing that an entry went stale is the point
// of inspecting the cache.
func (cache *DeviceCache) Items() []Item {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	now := time.Now()
	items := make([]Item, 0, len(cache.entries))

	for key, entry := range cache.entries {
		items = append(items, Item{
			Key:        key,
			Text:       entry.value.Text,
			RenderedAt: entry.value.RenderedAt,
			ExpiresAt:  entry.expiresAt,
			Expired:    !entry.infinite && now.After(entry.expiresAt),
			Forever:    entry.infinite,
		})
	}

	slices.SortFunc(items, func(a, b Item) int {
		return strings.Compare(a.Key, b.Key)
	})

	return items
}

func (cache *DeviceCache) EvictExpired() {
	now := time.Now()
	cache.mu.Lock()
	defer cache.mu.Unlock()

	for key, entry := range cache.entries {
		if entry.infinite {
			continue
		}

		if now.After(entry.expiresAt) {
			delete(cache.entries, key)
		}
	}
}
