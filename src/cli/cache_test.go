package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/daemon/ipc"

	"github.com/stretchr/testify/require"
)

func TestWriteCacheScopesGroupsAndRedacts(t *testing.T) {
	var out strings.Builder

	writeCacheScopes(&out, []*ipc.CacheScope{
		{
			Name: "device",
			Entries: []*ipc.CacheEntry{
				{Key: "copilot_token", Redacted: true},
				{Key: "is_wsl", Value: "false"},
			},
		},
		{Name: "session", Entries: nil},
		{
			Name:      "rendered segments",
			SessionId: "4242",
			Entries:   []*ipc.CacheEntry{{Key: "time", Value: "12:04"}},
		},
	})

	rendered := out.String()

	require.Contains(t, rendered, "<redacted>")
	require.NotContains(t, rendered, "ghu_", "a redacted entry must carry no value")
	require.Contains(t, rendered, "is_wsl")
	require.Contains(t, rendered, "device (2)")
	require.Contains(t, rendered, "rendered segments (1) · session 4242")
	require.NotContains(t, rendered, "session (0)", "an empty scope is not worth a heading")
}

func TestWriteCacheScopesSaysSoWhenEverythingIsEmpty(t *testing.T) {
	var out strings.Builder

	writeCacheScopes(&out, []*ipc.CacheScope{
		{Name: "device"},
		{Name: "session"},
	})

	require.Equal(t, "the cache is empty\n", out.String())
}

func TestCacheEntryLifetime(t *testing.T) {
	require.Equal(t, "never expires", cacheEntryLifetime(&ipc.CacheEntry{}))
	require.Equal(t, "expired", cacheEntryLifetime(&ipc.CacheEntry{Expired: true, Expires: 1}))
	require.Contains(t, cacheEntryLifetime(&ipc.CacheEntry{Expires: 1756000000}), "expires 20")
}

func TestCacheClearInitNoDaemonClearsInitScripts(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("PROMPTO_CACHE_DIR", tempDir)
	t.Setenv("XDG_RUNTIME_DIR", tempDir)
	t.Setenv("XDG_STATE_HOME", tempDir)
	cache.ResetPath()
	t.Cleanup(cache.ResetPath)

	cacheDir := cache.Path()
	initFile := filepath.Join(cacheDir, "init.test.zsh")
	require.NoError(t, os.WriteFile(initFile, []byte("echo test"), 0o644))

	clearInit = true
	t.Cleanup(func() {
		clearInit = false
	})

	var out strings.Builder
	cacheClearCmd.SetOut(&out)
	cacheClearCmd.Run(cacheClearCmd, nil)

	require.Equal(t, "init scripts cleared\n", out.String())
	_, err := os.Stat(initFile)
	require.True(t, os.IsNotExist(err), "init script should have been deleted")
}

func TestCacheClearNoDaemonFailsWithoutInit(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("PROMPTO_CACHE_DIR", tempDir)
	t.Setenv("XDG_RUNTIME_DIR", tempDir)
	t.Setenv("XDG_STATE_HOME", tempDir)
	cache.ResetPath()
	t.Cleanup(cache.ResetPath)

	cacheDir := cache.Path()
	initFile := filepath.Join(cacheDir, "init.test.zsh")
	require.NoError(t, os.WriteFile(initFile, []byte("echo test"), 0o644))

	clearInit = false
	exitcode = 0
	t.Cleanup(func() {
		exitcode = 0
	})

	var errOut strings.Builder
	cacheClearCmd.SetErr(&errOut)
	cacheClearCmd.Run(cacheClearCmd, nil)

	require.Contains(t, errOut.String(), errNoDaemon)
	require.Equal(t, 1, exitcode)

	_, err := os.Stat(initFile)
	require.NoError(t, err, "init script must NOT be deleted when --init is not passed")
}
