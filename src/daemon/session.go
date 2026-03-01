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

// SessionManager tracks active shell sessions by PID and emits lifecycle callbacks.
type SessionManager struct {
	sessions     map[int]*Session
	onUnregister func(int)
	onEmpty      func()
	mu           sync.RWMutex
}

func NewSessionManager(onUnregister func(int), onEmpty func()) *SessionManager {
	return &SessionManager{
		sessions:     make(map[int]*Session),
		onUnregister: onUnregister,
		onEmpty:      onEmpty,
	}
}

// Register adds the PID to tracked sessions and starts an exit watcher.
func (sm *SessionManager) Register(pid int, uuid, shell string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.sessions[pid]; exists {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	sm.sessions[pid] = &Session{
		PID:    pid,
		UUID:   uuid,
		Shell:  shell,
		cancel: cancel,
	}

	go sm.watchProcess(ctx, pid)
}

// Unregister removes the PID from tracked sessions.
func (sm *SessionManager) Unregister(pid int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.unregisterLocked(pid)
}

func (sm *SessionManager) unregisterLocked(pid int) {
	session, exists := sm.sessions[pid]
	if !exists {
		return
	}

	session.cancel()
	delete(sm.sessions, pid)

	if sm.onUnregister != nil {
		sm.onUnregister(pid)
	}

	if len(sm.sessions) != 0 {
		return
	}

	if sm.onEmpty != nil {
		sm.onEmpty()
	}
}

func (sm *SessionManager) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

func (sm *SessionManager) watchProcess(ctx context.Context, pid int) {
	waitForProcessExit(ctx, pid)

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.sessions[pid]; !exists {
		return
	}

	sm.unregisterLocked(pid)
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
