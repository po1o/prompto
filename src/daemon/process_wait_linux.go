//go:build linux

package daemon

import (
	"context"
	"errors"

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
		// Retry on EINTR: Go's runtime signals threads routinely (e.g. for
		// preemption), and an interrupted wait does not mean the process exited.
		for {
			_, err := unix.Poll(fds, -1)
			if !errors.Is(err, unix.EINTR) {
				return
			}
		}
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}
}
