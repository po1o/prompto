package daemon

import (
	"context"
	"sync"
	"time"
)

// Session tracks a watched shell process by PID.
type Session struct {
	cancel context.CancelFunc
	UUID   string
	Shell  string
	PID    int
}

// ProcessTracker tracks active shell sessions by PID and emits lifecycle callbacks.
type ProcessTracker struct {
	sessions     map[int]*Session
	onUnregister func(int)
	onEmpty      func()
	mu           sync.RWMutex
}

func NewProcessTracker(onUnregister func(int), onEmpty func()) *ProcessTracker {
	return &ProcessTracker{
		sessions:     make(map[int]*Session),
		onUnregister: onUnregister,
		onEmpty:      onEmpty,
	}
}

// Register adds the PID to tracked sessions and starts an exit watcher.
func (tracker *ProcessTracker) Register(pid int, uuid, shell string) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	if _, exists := tracker.sessions[pid]; exists {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	tracker.sessions[pid] = &Session{
		PID:    pid,
		UUID:   uuid,
		Shell:  shell,
		cancel: cancel,
	}

	go tracker.watchProcess(ctx, pid)
}

// Unregister removes the PID from tracked sessions.
func (tracker *ProcessTracker) Unregister(pid int) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.unregisterLocked(pid)
}

func (tracker *ProcessTracker) unregisterLocked(pid int) {
	session, exists := tracker.sessions[pid]
	if !exists {
		return
	}

	session.cancel()
	delete(tracker.sessions, pid)

	if tracker.onUnregister != nil {
		tracker.onUnregister(pid)
	}

	if len(tracker.sessions) != 0 {
		return
	}

	if tracker.onEmpty != nil {
		tracker.onEmpty()
	}
}

func (tracker *ProcessTracker) Count() int {
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()
	return len(tracker.sessions)
}

func (tracker *ProcessTracker) watchProcess(ctx context.Context, pid int) {
	waitForProcessExit(ctx, pid)

	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	if _, exists := tracker.sessions[pid]; !exists {
		return
	}

	tracker.unregisterLocked(pid)
}

func pollForProcessExit(ctx context.Context, pid int) {
	if !IsProcessRunning(pid) {
		return
	}

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !IsProcessRunning(pid) {
				return
			}
		}
	}
}
