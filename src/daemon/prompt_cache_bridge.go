package daemon

import (
	"time"

	"github.com/jandedobbeleer/oh-my-posh/src/color"
	"github.com/jandedobbeleer/oh-my-posh/src/prompt"
)

type promptDeviceCacheBridge struct {
	cache *DeviceCache
}

func newPromptDeviceCacheBridge(cache *DeviceCache) *promptDeviceCacheBridge {
	if cache == nil {
		return nil
	}

	return &promptDeviceCacheBridge{
		cache: cache,
	}
}

func (bridge *promptDeviceCacheBridge) Get(key string) (prompt.DeviceCacheEntry, bool) {
	if bridge == nil || bridge.cache == nil {
		return prompt.DeviceCacheEntry{}, false
	}

	value, ok := bridge.cache.Get(key)
	if !ok {
		return prompt.DeviceCacheEntry{}, false
	}

	return prompt.DeviceCacheEntry{
		RenderedAt: value.RenderedAt,
		Text:       value.Text,
		Foreground: color.Ansi(value.Foreground),
		Background: color.Ansi(value.Background),
	}, true
}

func (bridge *promptDeviceCacheBridge) Set(key string, value prompt.DeviceCacheEntry, ttl time.Duration) {
	if bridge == nil || bridge.cache == nil {
		return
	}

	bridge.cache.Set(key, SegmentRenderValue{
		RenderedAt: value.RenderedAt,
		Text:       value.Text,
		Foreground: string(value.Foreground),
		Background: string(value.Background),
	}, ttl)
}
