//go:build freebsd

package daemon

import (
	"context"
	"errors"
	"syscall"
)

// waitForProcessExit blocks until the process with the given PID exits.
func waitForProcessExit(ctx context.Context, pid int) {
	if !IsProcessRunning(pid) {
		return
	}

	kq, err := syscall.Kqueue()
	if err != nil {
		pollForProcessExit(ctx, pid)
		return
	}
	defer syscall.Close(kq)

	event := syscall.Kevent_t{
		Filter: syscall.EVFILT_PROC,
		Flags:  syscall.EV_ADD | syscall.EV_ONESHOT,
		Fflags: syscall.NOTE_EXIT,
	}
	setIdent(&event, pid)

	_, err = syscall.Kevent(kq, []syscall.Kevent_t{event}, nil, nil)
	if err != nil {
		pollForProcessExit(ctx, pid)
		return
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		events := make([]syscall.Kevent_t, 1)
		// Retry on EINTR: Go's runtime signals threads routinely (e.g. for
		// preemption), and an interrupted wait does not mean the process exited.
		for {
			_, err := syscall.Kevent(kq, nil, events, nil)
			if !errors.Is(err, syscall.EINTR) {
				return
			}
		}
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}
}
