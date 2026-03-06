package daemon

import (
	"sync"
	"time"
)

const DefaultDeviceCacheTTL = 7 * 24 * time.Hour

type SegmentRenderValue struct {
	RenderedAt time.Time
	Text       string
	Foreground string
	Background string
}

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
