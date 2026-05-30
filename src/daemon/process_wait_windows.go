//go:build windows

package daemon

import (
	"context"

	"golang.org/x/sys/windows"
)

// waitForProcessExit blocks until the process with the given PID exits.
func waitForProcessExit(ctx context.Context, pid int) {
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		if !IsProcessRunning(pid) {
			return
		}
		pollForProcessExit(ctx, pid)
		return
	}
	defer func() { _ = windows.CloseHandle(handle) }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = windows.WaitForSingleObject(handle, windows.INFINITE)
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}
}
