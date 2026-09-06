package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClearInitRemovesOnlyInitScripts(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("PROMPTO_CACHE_DIR", tempDir)
	ResetPath()
	t.Cleanup(ResetPath)

	cacheDir := Path()

	init1 := filepath.Join(cacheDir, "init.1234.zsh")
	init2 := filepath.Join(cacheDir, "init.5678.bash")
	otherFile := filepath.Join(cacheDir, "prompto.log")
	subDir := filepath.Join(cacheDir, "init.dir")

	require.NoError(t, os.WriteFile(init1, []byte("zsh"), 0o644))
	require.NoError(t, os.WriteFile(init2, []byte("bash"), 0o644))
	require.NoError(t, os.WriteFile(otherFile, []byte("log"), 0o644))
	require.NoError(t, os.Mkdir(subDir, 0o755))

	require.NoError(t, ClearInit())

	// init scripts should be removed
	_, err := os.Stat(init1)
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(init2)
	require.True(t, os.IsNotExist(err))

	// other files and directories must be preserved
	_, err = os.Stat(otherFile)
	require.NoError(t, err)
	_, err = os.Stat(subDir)
	require.NoError(t, err)
}
