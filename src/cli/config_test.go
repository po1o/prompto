package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	runtimelib "runtime"
	"testing"

	bundledthemes "github.com/po1o/prompto/src/themes"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveConfigPathPrefersExplicitFlag(t *testing.T) {
	previous := configFlag
	configFlag = filepath.Join(t.TempDir(), "explicit-config.yaml")

	t.Cleanup(func() {
		configFlag = previous
	})

	assert.Equal(t, filepath.Clean(configFlag), resolveConfigPath())
}

func TestResolveConfigPathUsesRunningDaemonConfigWhenNoFlag(t *testing.T) {
	previous := configFlag
	configFlag = ""
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)
	daemonConfigPath := filepath.Join(t.TempDir(), "daemon-config.yaml")

	lockDir := filepath.Join(stateHome, "prompto")
	err := os.MkdirAll(lockDir, 0o755)
	assert.NoError(t, err)

	lockPath := filepath.Join(lockDir, "daemon.lock")
	err = os.WriteFile(lockPath, fmt.Appendf(nil, "%d\n%s", os.Getpid(), daemonConfigPath), 0o600)
	assert.NoError(t, err)

	t.Cleanup(func() {
		configFlag = previous
	})

	assert.Equal(t, filepath.Clean(daemonConfigPath), resolveConfigPath())
}

func TestResolveConfigPathFallsBackToDefaultPath(t *testing.T) {
	previous := configFlag
	configFlag = ""
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	setDefaultConfigEnv(t)

	t.Cleanup(func() {
		configFlag = previous
	})

	assert.Equal(t, resolveDefaultConfigPath(), resolveConfigPath())
}

func TestFormatColumnsUsesTerminalWidth(t *testing.T) {
	items := []string{"alpha", "beta", "gamma", "delta"}
	output := formatColumns(items, 20)

	assert.Equal(t, "alpha  gamma\nbeta   delta\n", output)
}

func TestWriteBundledThemeWritesDefaultConfigPath(t *testing.T) {
	setDefaultConfigEnv(t)

	cmd := newTestConfigCommand()
	err := writeBundledTheme(cmd, "tokyo")
	require.NoError(t, err)

	expectedPath := resolveDefaultConfigPath()
	data, readErr := os.ReadFile(expectedPath)
	require.NoError(t, readErr)

	expectedTheme, ok := bundledthemes.Get("tokyo")
	require.True(t, ok)
	assert.Equal(t, expectedTheme, string(data))
}

func TestWriteBundledThemeRejectsOverwriteWithoutConfirmation(t *testing.T) {
	setDefaultConfigEnv(t)

	targetPath := resolveDefaultConfigPath()
	require.NoError(t, os.MkdirAll(filepath.Dir(targetPath), 0o755))
	require.NoError(t, os.WriteFile(targetPath, []byte("existing"), 0o644))

	cmd := newTestConfigCommand()
	cmd.SetIn(bytes.NewBufferString("n\n"))

	err := writeBundledTheme(cmd, "tokyo")
	require.Error(t, err)
	assert.EqualError(t, err, "aborted")

	data, readErr := os.ReadFile(targetPath)
	require.NoError(t, readErr)
	assert.Equal(t, "existing", string(data))
}

func TestWriteBundledThemeOverwritesAfterConfirmation(t *testing.T) {
	setDefaultConfigEnv(t)

	targetPath := resolveDefaultConfigPath()
	require.NoError(t, os.MkdirAll(filepath.Dir(targetPath), 0o755))
	require.NoError(t, os.WriteFile(targetPath, []byte("existing"), 0o644))

	cmd := newTestConfigCommand()
	cmd.SetIn(bytes.NewBufferString("yes\n"))

	err := writeBundledTheme(cmd, "tokyo")
	require.NoError(t, err)

	expectedTheme, ok := bundledthemes.Get("tokyo")
	require.True(t, ok)

	data, readErr := os.ReadFile(targetPath)
	require.NoError(t, readErr)
	assert.Equal(t, expectedTheme, string(data))
}

func newTestConfigCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	return cmd
}

func setDefaultConfigEnv(t *testing.T) {
	t.Helper()

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", configHome)

	if runtimelib.GOOS != "windows" {
		return
	}

	t.Setenv("APPDATA", configHome)
	t.Setenv("LOCALAPPDATA", configHome)
	t.Setenv("USERPROFILE", configHome)
}
