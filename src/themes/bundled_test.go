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

func TestBundledThemesParse(t *testing.T) {
	for _, name := range Names() {
		content, ok := Get(name)
		require.True(t, ok, name)

		_, err := config.ParseLayoutYAML([]byte(content))
		require.NoError(t, err, name)
	}
}
