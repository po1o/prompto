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
		if !isPIDAlive(pid) {
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
		for {
			_, pollErr := unix.Poll(fds, -1)
			if pollErr != unix.EINTR {
				return
			}
		}
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}
}
