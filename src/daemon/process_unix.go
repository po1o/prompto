//go:build !windows

package daemon

import (
	"os"
	"syscall"
)

func IsProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}

	return err == syscall.EPERM
}
