package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsConsole(t *testing.T) {
	t.Parallel()

	cases := []struct {
		env      map[string]string
		name     string
		expected bool
	}{
		{
			name:     "linux virtual console",
			env:      map[string]string{"TERM": "linux"},
			expected: true,
		},
		{
			name:     "terminal emulator",
			env:      map[string]string{"TERM": "xterm-256color"},
			expected: false,
		},
		{
			name:     "no TERM at all",
			env:      map[string]string{},
			expected: false,
		},
		{
			name:     "forced on overrides TERM",
			env:      map[string]string{"TERM": "xterm-256color", ConsoleEnv: "1"},
			expected: true,
		},
		{
			name:     "forced off overrides TERM",
			env:      map[string]string{"TERM": "linux", ConsoleEnv: "0"},
			expected: false,
		},
		{
			name:     "unrecognized override value falls back to TERM",
			env:      map[string]string{"TERM": "linux", ConsoleEnv: "yes"},
			expected: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			getenv := func(key string) string { return tc.env[key] }
			require.Equal(t, tc.expected, IsConsole(getenv))
		})
	}
}

func TestConsoleVariant(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "default config",
			input:    filepath.Join("home", "prompto", "config.yaml"),
			expected: filepath.Join("home", "prompto", "config.console.yaml"),
		},
		{
			name:     "yml extension is preserved",
			input:    filepath.Join("home", "config.yml"),
			expected: filepath.Join("home", "config.console.yml"),
		},
		{
			name:     "dots in the name only split on the extension",
			input:    filepath.Join("home", "my.theme.yaml"),
			expected: filepath.Join("home", "my.theme.console.yaml"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.expected, ConsoleVariant(tc.input))
		})
	}
}

func TestResolveUsesConsoleVariantOnlyWhenItExists(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	consolePath := filepath.Join(dir, "config.console.yaml")

	require.NoError(t, os.WriteFile(configPath, []byte("prompt: []\n"), 0o644))

	t.Setenv("TERM", "linux")

	// No variant on disk yet: the requested config is kept.
	require.Equal(t, configPath, Resolve(configPath))

	require.NoError(t, os.WriteFile(consolePath, []byte("prompt: []\n"), 0o644))

	require.Equal(t, consolePath, Resolve(configPath))
}

func TestResolveIgnoresConsoleVariantOffConsole(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	consolePath := filepath.Join(dir, "config.console.yaml")

	require.NoError(t, os.WriteFile(configPath, []byte("prompt: []\n"), 0o644))
	require.NoError(t, os.WriteFile(consolePath, []byte("prompt: []\n"), 0o644))

	t.Setenv("TERM", "xterm-256color")

	require.Equal(t, configPath, Resolve(configPath))
}

// A console config passed explicitly must not send us looking for
// config.console.console.yaml.
func TestResolveIsIdempotentOnAConsoleConfig(t *testing.T) {
	dir := t.TempDir()
	consolePath := filepath.Join(dir, "config.console.yaml")

	require.NoError(t, os.WriteFile(consolePath, []byte("prompt: []\n"), 0o644))

	t.Setenv("TERM", "linux")

	require.Equal(t, consolePath, Resolve(consolePath))
}

func TestResolveFallsBackToDefaultPath(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")

	require.Equal(t, DefaultPath(), Resolve(""))
}
