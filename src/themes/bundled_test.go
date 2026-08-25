package themes

import (
	"testing"

	"github.com/po1o/prompto/src/config"
	"github.com/stretchr/testify/require"
)

func TestGetStripsThemeSuffix(t *testing.T) {
	content, ok := Get("tokyo.prompto.yaml")
	require.True(t, ok)
	require.NotEmpty(t, content)
}

func TestNamesIncludesPoloTheme(t *testing.T) {
	names := Names()
	require.Contains(t, names, "polo")
}

func TestConsoleVariantsAreNotListedAsThemes(t *testing.T) {
	for _, name := range Names() {
		require.NotContains(t, name, ".console", "console variants belong to their base theme")
	}
}

func TestGetConsoleReturnsPoloVariant(t *testing.T) {
	content, ok := GetConsole("polo")
	require.True(t, ok)
	require.NotEmpty(t, content)

	main, ok := Get("polo")
	require.True(t, ok)
	require.NotEqual(t, main, content)
}

func TestGetConsoleMissesThemesWithoutVariant(t *testing.T) {
	_, ok := GetConsole("tokyo")
	require.False(t, ok)
}

// Every console variant must belong to a real theme, or `config set` could
// never reach it.
func TestConsoleVariantsHaveABaseTheme(t *testing.T) {
	for name := range bundledConsoleThemes {
		_, ok := Get(name)
		require.True(t, ok, name)
	}
}

func TestBundledConsoleThemesParse(t *testing.T) {
	for name := range bundledConsoleThemes {
		content, ok := GetConsole(name)
		require.True(t, ok, name)

		_, err := config.ParseLayoutYAML([]byte(content))
		require.NoError(t, err, name)
	}
}

func TestBundledThemesParse(t *testing.T) {
	for _, name := range Names() {
		content, ok := Get(name)
		require.True(t, ok, name)

		_, err := config.ParseLayoutYAML([]byte(content))
		require.NoError(t, err, name)
	}
}
