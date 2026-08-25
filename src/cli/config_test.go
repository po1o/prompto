package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/po1o/prompto/src/config"
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
	setDaemonStateEnv(t, stateHome)
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

func TestWriteBundledThemeInstallsConsoleVariantAlongside(t *testing.T) {
	setDefaultConfigEnv(t)

	cmd := newTestConfigCommand()
	require.NoError(t, writeBundledTheme(cmd, "polo"))

	targetPath := resolveDefaultConfigPath()
	consolePath := config.ConsoleVariant(targetPath)

	expectedTheme, ok := bundledthemes.Get("polo")
	require.True(t, ok)
	data, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	assert.Equal(t, expectedTheme, string(data))

	expectedConsole, ok := bundledthemes.GetConsole("polo")
	require.True(t, ok)
	consoleData, err := os.ReadFile(consolePath)
	require.NoError(t, err)
	assert.Equal(t, expectedConsole, string(consoleData))
}

// A theme without a console variant must not leave a stray config.console.yaml.
func TestWriteBundledThemeWithoutConsoleVariantWritesOnlyMainConfig(t *testing.T) {
	setDefaultConfigEnv(t)

	_, ok := bundledthemes.GetConsole("tokyo")
	require.False(t, ok, "tokyo is expected to have no console variant")

	cmd := newTestConfigCommand()
	require.NoError(t, writeBundledTheme(cmd, "tokyo"))

	assert.NoFileExists(t, config.ConsoleVariant(resolveDefaultConfigPath()))
}

// Switching to a theme with no console variant leaves the previous one in
// place, so the user has to be told it still applies on the console.
func TestWriteBundledThemeWarnsAboutStaleConsoleVariant(t *testing.T) {
	setDefaultConfigEnv(t)

	targetPath := resolveDefaultConfigPath()
	consolePath := config.ConsoleVariant(targetPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(targetPath), 0o755))
	require.NoError(t, os.WriteFile(consolePath, []byte("left over"), 0o644))

	cmd := newTestConfigCommand()
	cmd.SetIn(bytes.NewBufferString("y\n"))

	require.NoError(t, writeBundledTheme(cmd, "tokyo"))

	stderr, ok := cmd.ErrOrStderr().(*bytes.Buffer)
	require.True(t, ok)
	assert.Contains(t, stderr.String(), consolePath)
	assert.Contains(t, stderr.String(), "left over from another config")

	// The warning must not double as permission to delete it.
	data, err := os.ReadFile(consolePath)
	require.NoError(t, err)
	assert.Equal(t, "left over", string(data))
}

// Declining the prompt must leave both files untouched rather than replacing
// config.yaml and stranding a stale console variant next to it.
func TestWriteBundledThemeAbortLeavesConsoleVariantUntouched(t *testing.T) {
	setDefaultConfigEnv(t)

	targetPath := resolveDefaultConfigPath()
	consolePath := config.ConsoleVariant(targetPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(targetPath), 0o755))
	require.NoError(t, os.WriteFile(targetPath, []byte("existing"), 0o644))
	require.NoError(t, os.WriteFile(consolePath, []byte("existing console"), 0o644))

	cmd := newTestConfigCommand()
	cmd.SetIn(bytes.NewBufferString("n\n"))

	err := writeBundledTheme(cmd, "polo")
	require.Error(t, err)
	assert.EqualError(t, err, "aborted")

	data, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	assert.Equal(t, "existing", string(data))

	consoleData, err := os.ReadFile(consolePath)
	require.NoError(t, err)
	assert.Equal(t, "existing console", string(consoleData))
}

// Only the console variant pre-exists: it still has to be confirmed, since
// installing would overwrite it.
func TestWriteBundledThemeConfirmsWhenOnlyConsoleVariantExists(t *testing.T) {
	setDefaultConfigEnv(t)

	targetPath := resolveDefaultConfigPath()
	consolePath := config.ConsoleVariant(targetPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(targetPath), 0o755))
	require.NoError(t, os.WriteFile(consolePath, []byte("existing console"), 0o644))

	cmd := newTestConfigCommand()
	cmd.SetIn(bytes.NewBufferString("n\n"))

	err := writeBundledTheme(cmd, "polo")
	require.Error(t, err)
	assert.EqualError(t, err, "aborted")

	assert.NoFileExists(t, targetPath)

	consoleData, err := os.ReadFile(consolePath)
	require.NoError(t, err)
	assert.Equal(t, "existing console", string(consoleData))
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
}

func setDaemonStateEnv(t *testing.T, stateHome string) {
	t.Helper()

	t.Setenv("XDG_STATE_HOME", stateHome)
}
