package daemon

import (
	"os"
	"path/filepath"
	libruntime "runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBinaryWatcherTriggersOnWrite(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "oh-my-posh")
	require.NoError(t, os.WriteFile(binaryPath, []byte("v1"), 0o755))

	triggered := make(chan struct{}, 1)
	watcher, err := newBinaryWatcher(binaryPath, func() {
		select {
		case triggered <- struct{}{}:
		default:
		}
	}, 25*time.Millisecond)
	require.NoError(t, err)
	t.Cleanup(func() { _ = watcher.Close() })

	require.NoError(t, os.WriteFile(binaryPath, []byte("v2"), 0o755))

	require.Eventually(t, func() bool {
		select {
		case <-triggered:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
}

func TestBinaryWatcherTriggersOnRemove(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "oh-my-posh")
	require.NoError(t, os.WriteFile(binaryPath, []byte("v1"), 0o755))

	triggered := make(chan struct{}, 1)
	watcher, err := newBinaryWatcher(binaryPath, func() {
		select {
		case triggered <- struct{}{}:
		default:
		}
	}, 25*time.Millisecond)
	require.NoError(t, err)
	t.Cleanup(func() { _ = watcher.Close() })

	require.NoError(t, os.Remove(binaryPath))

	require.Eventually(t, func() bool {
		select {
		case <-triggered:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
}

func TestBinaryWatcherTracksResolvedPathForSymlinkInput(t *testing.T) {
	if libruntime.GOOS == windowsOS {
		t.Skip("symlink behavior differs on windows")
	}

	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))

	realBinaryV1 := filepath.Join(tmpDir, "oh-my-posh-v1")
	require.NoError(t, os.WriteFile(realBinaryV1, []byte("v1"), 0o755))

	symlinkPath := filepath.Join(binDir, "oh-my-posh")
	require.NoError(t, os.Symlink(realBinaryV1, symlinkPath))

	triggered := make(chan struct{}, 1)
	watcher, err := newBinaryWatcher(symlinkPath, func() {
		select {
		case triggered <- struct{}{}:
		default:
		}
	}, 25*time.Millisecond)
	require.NoError(t, err)
	t.Cleanup(func() { _ = watcher.Close() })

	require.NoError(t, os.WriteFile(realBinaryV1, []byte("v2"), 0o755))

	require.Eventually(t, func() bool {
		select {
		case <-triggered:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
}
