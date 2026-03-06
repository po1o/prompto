package daemon

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jandedobbeleer/oh-my-posh/src/log"
)

// LockFile represents an exclusive lock file to prevent multiple daemon instances.
type LockFile struct {
	path string
}

// NewLockFile creates and acquires an exclusive lock file.
// Returns error if lock is already held by another process.
func NewLockFile(configPath string) (*LockFile, error) {
	stateDir := statePath()

	// Ensure state directory exists
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	path := filepath.Join(stateDir, "daemon.lock")

	// Try to acquire lock
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		// Lock file exists - check if process is still alive
		if os.IsExist(err) {
			pid, _, pidErr := ReadLockInfo(path)
			if pidErr != nil {
				// Can't read PID - remove stale lock and retry
				log.Debug("failed to read PID from lock file, removing stale lock")
				_ = os.Remove(path)
				return NewLockFile(configPath)
			}

			if !IsProcessRunning(pid) {
				// Process is dead - remove stale lock and retry
				log.Debugf("daemon process %d is not running, removing stale lock", pid)
				_ = os.Remove(path)
				return NewLockFile(configPath)
			}

			return nil, fmt.Errorf("daemon already running with PID %d", pid)
		}

		return nil, err
	}

	lf := &LockFile{
		path: path,
	}

	// Write our PID and configPath to the lock file
	if err := lf.WriteLockInfo(file, configPath); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, err
	}
	_ = file.Close()

	return lf, nil
}

// WriteLockInfo writes the daemon PID and config path to the lock file.
func (lf *LockFile) WriteLockInfo(file *os.File, configPath string) error {
	_, err := fmt.Fprintf(file, "%d\n%s", os.Getpid(), configPath)
	if err != nil {
		return err
	}
	return file.Sync()
}

// ReadLockInfo reads the PID and config path from an existing lock file.
func ReadLockInfo(path string) (int, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Read PID (first line)
	if !scanner.Scan() {
		return 0, "", fmt.Errorf("empty lock file")
	}
	pidStr := strings.TrimSpace(scanner.Text())
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, "", fmt.Errorf("invalid PID in lock file: %s", pidStr)
	}

	// Read Config Path (second line)
	var configPath string
	if scanner.Scan() {
		configPath = strings.TrimSpace(scanner.Text())
	}

	return pid, configPath, nil
}

// ReadPID reads the PID from an existing lock file.
func ReadPID(path string) (int, error) {
	pid, _, err := ReadLockInfo(path)
	return pid, err
}

// Release removes the lock file.
func (lf *LockFile) Release() error {
	return os.Remove(lf.path)
}

// CleanupLock removes a stale lock file (for crash recovery).
func CleanupLock() error {
	path := filepath.Join(statePath(), "daemon.lock")
	return os.Remove(path)
}

// GetRunningDaemonInfo returns the PID and config path of the running daemon.
func GetRunningDaemonInfo() (int, string, error) {
	path := filepath.Join(statePath(), "daemon.lock")
	return ReadLockInfo(path)
}

// KillDaemon checks for an existing daemon and kills it if running.
// It also removes the lock file.
func KillDaemon() error {
	path := filepath.Join(statePath(), "daemon.lock")

	pid, err := ReadPID(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		// Corrupt lock file, force remove
		return os.Remove(path)
	}

	if IsProcessRunning(pid) {
		proc, err := os.FindProcess(pid)
		if err == nil {
			_ = proc.Kill()
			_, _ = proc.Wait()
		}
	}

	return os.Remove(path)
}
