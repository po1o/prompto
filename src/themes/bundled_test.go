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

// Every bundled theme ships a variant, so the miss has to be staged.
func TestGetConsoleMissesThemesWithoutVariant(t *testing.T) {
	WithoutConsoleVariant("tokyo", func() {
		_, ok := GetConsole("tokyo")
		require.False(t, ok)
	})

	_, ok := GetConsole("tokyo")
	require.True(t, ok, "the variant must come back afterwards")
}

// A console session falls back to the main config, so every theme having a
// variant is what makes the console usable across the board.
func TestEveryThemeShipsAConsoleVariant(t *testing.T) {
	for _, name := range Names() {
		_, ok := GetConsole(name)
		require.True(t, ok, name)
	}
}

// consoleSafeRunes are the non-ASCII characters a console variant may still
// use. The Linux console font carries the CP437 arrows, and they read better
// than the ASCII stand-ins for git's ahead/behind counts; nothing else outside
// ASCII can be relied on.
var consoleSafeRunes = map[rune]bool{
	'\u2191': true, // up arrow
	'\u2193': true, // down arrow
}

// A console variant that reaches for a glyph the console font lacks defeats its
// own purpose: the reader gets a blank or a box where the prompt should be.
func TestConsoleVariantsUseConsoleSafeCharacters(t *testing.T) {
	for name, content := range bundledConsoleThemes {
		for _, r := range content {
			if r < 0x80 || consoleSafeRunes[r] {
				continue
			}

			require.Fail(t, "unsafe character", "%s contains %U", name, r)
		}
	}
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
