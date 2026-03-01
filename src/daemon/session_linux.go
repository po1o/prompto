//go:build linux

package daemon

import (
	"context"

	"golang.org/x/sys/unix"
)

// waitForProcessExit blocks until the process with the given PID exits.
func waitForProcessExit(ctx context.Context, pid int) {
	pidfd, err := unix.PidfdOpen(pid, 0)
	if err != nil {
		if !IsProcessRunning(pid) {
			return
		}
		pollForProcessExit(ctx, pid)
		return
	}
	defer unix.Close(pidfd)

	done := make(chan struct{})
	go func() {
		defer close(done)
		fds := []unix.PollFd{{Fd: int32(pidfd), Events: unix.POLLIN}}
		_, _ = unix.Poll(fds, -1)
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}
}
