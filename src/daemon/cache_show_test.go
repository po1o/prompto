package daemon

import (
	"testing"
	"time"

	"github.com/po1o/prompto/src/cache"

	"github.com/stretchr/testify/require"
)

func TestIsCredentialKey(t *testing.T) {
	cases := []struct {
		key      string
		redacted bool
	}{
		// Every credential the tree caches today.
		{"copilot_token", true},
		{"ytmda_token", true},
		{"withings_access_token", true},
		{"withings_refresh_token", true},
		{"strava_access_token", true},
		{"strava_refresh_token", true},
		// Shapes a future segment might plausibly use.
		{"SOME_API_KEY", true},
		{"api-key", true},
		{"client_secret", true},
		{"user_password", true},
		{"aws_credentials", true},
		{"aws-access-key", true},
		{"ssh-private-key", true},
		// Values that are safe to print, including one that merely looks close.
		{"is_wsl", false},
		{"environment_platform", false},
		{"ttl", false},
		{"toggle_cache", false},
		{"daemon_cache_go_/home/user", false},
		{"keyboard_layout", false},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			require.Equal(t, tc.redacted, IsCredentialKey(tc.key))
		})
	}
}

// cache show renders whatever the daemon holds straight to a terminal, and the
// device store holds OAuth tokens. The value must never leave the daemon.
func TestCacheScopesRedactsCredentialValues(t *testing.T) {
	cache.Delete(cache.Device, "copilot_token")
	cache.Delete(cache.Device, "is_wsl")
	t.Cleanup(func() {
		cache.Delete(cache.Device, "copilot_token")
		cache.Delete(cache.Device, "is_wsl")
	})

	cache.Set(cache.Device, "copilot_token", "ghu_supersecretvalue", cache.INFINITE)
	cache.Set(cache.Device, "is_wsl", false, cache.INFINITE)

	daemon := New(&rendererStub{})

	items := scopeItems(t, daemon.CacheScopes(""), DeviceScopeName, "")

	token := itemByKey(t, items, "copilot_token")
	require.True(t, token.Redacted)
	require.Empty(t, token.Value, "a credential value must not reach the client")

	wsl := itemByKey(t, items, "is_wsl")
	require.False(t, wsl.Redacted)
	require.Equal(t, "false", wsl.Value)
}

func TestCacheScopesReportsEveryStore(t *testing.T) {
	daemon := New(&rendererStub{})

	names := make(map[string]bool)
	for _, scope := range daemon.CacheScopes("") {
		names[scope.Name] = true
	}

	require.True(t, names[DeviceScopeName])
	require.True(t, names[SessionScopeName])
	require.True(t, names[SegmentsScopeName])
}

// A session id narrows the per-session scopes to one shell; the process-wide
// stores are not per-session and are always reported.
func TestCacheScopesWithASessionIDKeepsTheSharedStores(t *testing.T) {
	daemon := New(&rendererStub{})

	for _, scope := range daemon.CacheScopes("does-not-exist") {
		require.Empty(t, scope.SessionID, "no session scope should survive an unmatched filter")
	}
}

func TestStoreItemsCarriesExpiry(t *testing.T) {
	cache.Delete(cache.Device, "forever")
	cache.Delete(cache.Device, "temporary")
	t.Cleanup(func() {
		cache.Delete(cache.Device, "forever")
		cache.Delete(cache.Device, "temporary")
	})

	cache.Set(cache.Device, "forever", "a", cache.INFINITE)
	cache.Set(cache.Device, "temporary", "b", cache.ONEDAY)

	items := storeItems(cache.Device)

	forever := itemByKey(t, items, "forever")
	require.True(t, forever.Expires.IsZero(), "an infinite entry has no expiry")
	require.False(t, forever.Expired)

	temporary := itemByKey(t, items, "temporary")
	require.False(t, temporary.Expires.IsZero())
	require.WithinDuration(t, time.Now().Add(24*time.Hour), temporary.Expires, time.Minute)
}

func scopeItems(t *testing.T, scopes []CacheScope, name, sessionID string) []CacheItem {
	t.Helper()

	for _, scope := range scopes {
		if scope.Name == name && scope.SessionID == sessionID {
			return scope.Items
		}
	}

	t.Fatalf("scope %q (session %q) not found", name, sessionID)
	return nil
}

func itemByKey(t *testing.T, items []CacheItem, key string) CacheItem {
	t.Helper()

	for _, item := range items {
		if item.Key == key {
			return item
		}
	}

	t.Fatalf("key %q not found", key)
	return CacheItem{}
}
