package daemon

import (
	"slices"
	"strings"
	"time"

	"github.com/po1o/prompto/src/cache"
)

// Names of the caches as `prompto cache show` reports them.
const (
	DeviceScopeName   = "device"
	SessionScopeName  = "session"
	SegmentsScopeName = "rendered segments"
)

// credentialKeyParts mark a cached value as a secret. The test is on the key
// rather than a list of known keys: the device cache already holds OAuth
// access and refresh tokens for several segments, and a segment that starts
// caching a token tomorrow is covered without anyone remembering this file.
var credentialKeyParts = []string{
	"token", "secret", "password", "credential",
	"api_key", "apikey", "access_key", "private_key", "secret_key",
}

// IsCredentialKey reports whether a cache key names something that must not be
// printed. `prompto cache show` renders whatever the daemon holds, and that
// includes credentials which would otherwise land in a terminal scrollback.
func IsCredentialKey(key string) bool {
	normalized := strings.ReplaceAll(strings.ToLower(key), "-", "_")

	return slices.ContainsFunc(credentialKeyParts, func(part string) bool {
		return strings.Contains(normalized, part)
	})
}

// CacheItem is one cached value, ready to render.
type CacheItem struct {
	Created  time.Time
	Expires  time.Time
	Key      string
	Value    string
	Type     string
	Expired  bool
	Redacted bool
}

// CacheScope groups cache items by where they are held.
type CacheScope struct {
	Name      string
	SessionID string
	Items     []CacheItem
}

// CacheScopes describes every cache the daemon holds.
//
// There are three, and they are not interchangeable, which is why the CLI
// prints them apart: the process-wide device and session stores that segments
// write through src/cache, and the rendered-segment cache. Session-scoped
// rendered segments live on each shell's own engine, so they are reported once
// per session; passing a sessionID narrows those to the caller's own shell.
func (daemon *Daemon) CacheScopes(sessionID string) []CacheScope {
	scopes := []CacheScope{
		{Name: DeviceScopeName, Items: storeItems(cache.Device)},
		{Name: SessionScopeName, Items: storeItems(cache.Session)},
		{Name: SegmentsScopeName, Items: daemon.deviceCacheItems()},
	}

	return append(scopes, daemon.sessionCacheScopes(sessionID)...)
}

// storeItems converts one of the process-wide stores, redacting credentials.
func storeItems(store cache.Store) []CacheItem {
	stored := cache.Items(store)
	items := make([]CacheItem, 0, len(stored))

	for _, entry := range stored {
		item := CacheItem{
			Key:      entry.Key,
			Value:    entry.Value,
			Type:     entry.Type,
			Created:  entry.CreatedAt,
			Expires:  entry.ExpiresAt,
			Expired:  entry.Expired,
			Redacted: IsCredentialKey(entry.Key),
		}

		if item.Redacted {
			item.Value = ""
		}

		items = append(items, item)
	}

	return items
}

func (daemon *Daemon) deviceCacheItems() []CacheItem {
	if daemon.deviceCache == nil {
		return nil
	}

	stored := daemon.deviceCache.Items()
	items := make([]CacheItem, 0, len(stored))

	for _, entry := range stored {
		items = append(items, CacheItem{
			Key:     entry.Key,
			Value:   entry.Text,
			Type:    "segment",
			Created: entry.RenderedAt,
			Expires: entry.ExpiresAt,
			Expired: entry.Expired,
		})
	}

	return items
}

// sessionCacheScopes reports the session-scoped rendered segments each live
// shell holds on its own engine. An empty sessionID reports every session.
func (daemon *Daemon) sessionCacheScopes(sessionID string) []CacheScope {
	if daemon.pipeline == nil || daemon.pipeline.registry == nil {
		return nil
	}

	engines := daemon.pipeline.registry.SessionEngines()
	scopes := make([]CacheScope, 0, len(engines))

	for id, engine := range engines {
		if sessionID != "" && id != sessionID {
			continue
		}

		stored := engine.SessionCacheItems()
		if len(stored) == 0 {
			continue
		}

		items := make([]CacheItem, 0, len(stored))
		for _, entry := range stored {
			items = append(items, CacheItem{
				Key:     entry.Key,
				Value:   entry.Text,
				Type:    "segment",
				Created: entry.RenderedAt,
			})
		}

		scopes = append(scopes, CacheScope{
			Name:      SegmentsScopeName,
			SessionID: id,
			Items:     items,
		})
	}

	slices.SortFunc(scopes, func(a, b CacheScope) int {
		return strings.Compare(a.SessionID, b.SessionID)
	})

	return scopes
}
