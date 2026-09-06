package shell

import (
	"os"
	"testing"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/runtime"

	"github.com/stretchr/testify/require"
)

func TestCacheValueChangesWhenDaemonModeChanges(t *testing.T) {
	flagsA := &runtime.Flags{
		Shell:      ZSH,
		ConfigHash: 42,
		Daemon:     false,
		Strict:     false,
	}
	envA := &runtime.Terminal{}
	envA.Init(flagsA)

	flagsB := &runtime.Flags{
		Shell:      ZSH,
		ConfigHash: 42,
		Daemon:     true,
		Strict:     false,
	}
	envB := &runtime.Terminal{}
	envB.Init(flagsB)

	if cacheValue(envA) == cacheValue(envB) {
		t.Fatalf("expected cache value to differ when daemon mode changes")
	}
}

func TestCacheValueChangesWhenStrictModeChanges(t *testing.T) {
	flagsA := &runtime.Flags{
		Shell:      ZSH,
		ConfigHash: 42,
		Daemon:     false,
		Strict:     false,
	}
	envA := &runtime.Terminal{}
	envA.Init(flagsA)

	flagsB := &runtime.Flags{
		Shell:      ZSH,
		ConfigHash: 42,
		Daemon:     false,
		Strict:     true,
	}
	envB := &runtime.Terminal{}
	envB.Init(flagsB)

	if cacheValue(envA) == cacheValue(envB) {
		t.Fatalf("expected cache value to differ when strict mode changes")
	}
}

func TestCacheValueChangesWhenConfigPathChanges(t *testing.T) {
	flagsA := &runtime.Flags{
		Shell:      ZSH,
		ConfigPath: "/tmp/a.omp.yaml",
		ConfigHash: 42,
	}
	envA := &runtime.Terminal{}
	envA.Init(flagsA)

	flagsB := &runtime.Flags{
		Shell:      ZSH,
		ConfigPath: "/tmp/b.omp.yaml",
		ConfigHash: 42,
	}
	envB := &runtime.Terminal{}
	envB.Init(flagsB)

	if cacheValue(envA) == cacheValue(envB) {
		t.Fatalf("expected cache value to differ when init config path changes")
	}
}

func TestHasScriptAndWriteScript(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("PROMPTO_CACHE_DIR", tempDir)
	cache.ResetPath()
	t.Cleanup(cache.ResetPath)

	scriptPathCache = ""
	t.Cleanup(func() {
		scriptPathCache = ""
	})

	flags := &runtime.Flags{
		Shell:      ZSH,
		ConfigHash: 12345,
	}
	env := &runtime.Terminal{}
	env.Init(flags)

	// Initially, no script exists.
	path, ok := hasScript(env)
	if ok || path != "" {
		t.Fatalf("expected hasScript to return false when script does not exist, got %v, %q", ok, path)
	}

	// Write script.
	writtenPath, err := writeScript(env, "echo 'hello zsh'")
	if err != nil {
		t.Fatalf("unexpected error writing script: %v", err)
	}

	// Now hasScript should return true with the written path.
	scriptPathCache = ""
	path, ok = hasScript(env)
	if !ok || path != writtenPath {
		t.Fatalf("expected hasScript to return true with path %q, got %v, %q", writtenPath, ok, path)
	}

	// Context change (different config hash) should cause hasScript to return false.
	flagsChanged := &runtime.Flags{
		Shell:      ZSH,
		ConfigHash: 99999,
	}
	envChanged := &runtime.Terminal{}
	envChanged.Init(flagsChanged)

	scriptPathCache = ""
	_, ok = hasScript(envChanged)
	if ok {
		t.Fatalf("expected hasScript to return false when context changes")
	}
}

func TestHasScriptCorruptedOrEmptyFile(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"empty file", ""},
		{"newline only", "\n"},
		{"corrupted signature", "# prompto-sig: invalid_hash\n"},
		{"wrong prefix", "echo 'hello'\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Setenv("PROMPTO_CACHE_DIR", tempDir)
			cache.ResetPath()
			t.Cleanup(cache.ResetPath)

			scriptPathCache = ""
			t.Cleanup(func() {
				scriptPathCache = ""
			})

			flags := &runtime.Flags{
				Shell:      ZSH,
				ConfigHash: 12345,
			}
			env := &runtime.Terminal{}
			env.Init(flags)

			path := scriptPath(env)
			require.NoError(t, os.WriteFile(path, []byte(tc.content), 0o644))

			scriptPathCache = ""
			gotPath, ok := hasScript(env)
			require.False(t, ok)
			require.Empty(t, gotPath)
		})
	}
}
