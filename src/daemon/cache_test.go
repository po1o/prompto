package daemon

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDeviceCacheSetGet(t *testing.T) {
	cache := NewDeviceCache()
	value := SegmentRenderValue{
		Text:       "segment",
		Foreground: "red",
		Background: "blue",
		RenderedAt: time.Now(),
	}

	cache.Set("path.main", value, time.Minute)
	got, ok := cache.Get("path.main")
	require.True(t, ok)
	require.Equal(t, value, got)
}

func TestDeviceCacheDefaultTTLIsUsedWhenTTLIsNonPositive(t *testing.T) {
	cache := NewDeviceCache()
	cache.SetDefaultTTL(30 * time.Millisecond)
	cache.Set("path.main", SegmentRenderValue{Text: "x"}, 0)

	time.Sleep(50 * time.Millisecond)

	_, ok := cache.Get("path.main")
	require.False(t, ok)
}

func TestDeviceCacheDeleteAndClear(t *testing.T) {
	cache := NewDeviceCache()
	cache.Set("a", SegmentRenderValue{Text: "a"}, time.Minute)
	cache.Set("b", SegmentRenderValue{Text: "b"}, time.Minute)

	cache.Delete("a")
	_, ok := cache.Get("a")
	require.False(t, ok)
	require.Equal(t, 1, cache.Count())

	cache.Clear()
	require.Equal(t, 0, cache.Count())
}

func TestDeviceCacheEvictExpired(t *testing.T) {
	cache := NewDeviceCache()
	cache.Set("short", SegmentRenderValue{Text: "short"}, 20*time.Millisecond)
	cache.Set("long", SegmentRenderValue{Text: "long"}, time.Minute)

	time.Sleep(40 * time.Millisecond)
	cache.EvictExpired()

	_, ok := cache.Get("short")
	require.False(t, ok)
	longValue, ok := cache.Get("long")
	require.True(t, ok)
	require.Equal(t, "long", longValue.Text)
}

func TestDeviceCacheConcurrentAccess(t *testing.T) {
	cache := NewDeviceCache()
	var wg sync.WaitGroup

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			key := "k"
			if index%2 == 0 {
				key = "k2"
			}
			cache.Set(key, SegmentRenderValue{Text: "v"}, time.Minute)
			cache.Get(key)
		}(i)
	}

	wg.Wait()
	require.GreaterOrEqual(t, cache.Count(), 1)
}
