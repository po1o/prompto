package daemon

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jandedobbeleer/oh-my-posh/src/prompt"
)

type Daemon struct {
	service     *Service
	sessions    *SessionManager
	idleTimeout time.Duration
	stopped     atomic.Bool
	idleToken   uint64
	mu          sync.Mutex
}

func New(renderer promptBundleRenderer) *Daemon {
	return NewWithIdleTimeout(5*time.Minute, renderer)
}

func NewWithIdleTimeout(idleTimeout time.Duration, renderer promptBundleRenderer) *Daemon {
	registry := NewEngineRegistry(prompt.New)
	gate := NewReloadGate()
	daemon := &Daemon{
		service:     NewService(registry, gate, renderer),
		idleTimeout: idleTimeout,
	}
	daemon.sessions = NewSessionManager(daemon.onSessionUnregister, daemon.onAllSessionsEnded)

	daemon.mu.Lock()
	daemon.scheduleIdleStopLocked()
	daemon.mu.Unlock()

	return daemon
}

func (daemon *Daemon) StartRender(request RenderRequest) RenderResponse {
	if daemon.stopped.Load() {
		return RenderResponse{Type: "stopped"}
	}

	daemon.registerSessionPID(request)

	return daemon.service.StartRender(request)
}

func (daemon *Daemon) NextUpdate(ctx context.Context, sessionID string, after uint64) (RenderResponse, bool) {
	if daemon.stopped.Load() {
		return RenderResponse{}, false
	}

	return daemon.service.NextUpdate(ctx, sessionID, after)
}

func (daemon *Daemon) CompleteSession(sessionID string) {
	if daemon.stopped.Load() {
		return
	}

	daemon.service.CompleteSession(sessionID)

	pid, ok := parseSessionPID(sessionID)
	if ok {
		daemon.sessions.Unregister(pid)
		return
	}

	if daemon.sessions.Count() != 0 {
		return
	}

	daemon.mu.Lock()
	daemon.scheduleIdleStopLocked()
	daemon.mu.Unlock()
}

func (daemon *Daemon) Reload(action func()) {
	if daemon.stopped.Load() {
		return
	}

	daemon.service.Reload(action)
}

func (daemon *Daemon) Snapshot() (active int, reloading bool) {
	return daemon.service.Snapshot()
}

func (daemon *Daemon) SessionCount() int {
	return daemon.service.SessionCount()
}

func (daemon *Daemon) SessionHub(sessionID string) *SessionUpdateHub {
	return daemon.service.SessionHub(sessionID)
}

func (daemon *Daemon) Stop() {
	if !daemon.stopped.CompareAndSwap(false, true) {
		return
	}

	daemon.mu.Lock()
	daemon.cancelIdleStopLocked()
	daemon.mu.Unlock()
}

func (daemon *Daemon) cancelIdleStopLocked() {
	daemon.idleToken++
}

func (daemon *Daemon) scheduleIdleStopLocked() {
	if daemon.idleTimeout <= 0 {
		return
	}

	daemon.idleToken++
	token := daemon.idleToken
	timeout := daemon.idleTimeout

	time.AfterFunc(timeout, func() {
		daemon.mu.Lock()
		if daemon.stopped.Load() || daemon.idleToken != token {
			daemon.mu.Unlock()
			return
		}
		daemon.mu.Unlock()

		if daemon.sessions.Count() == 0 {
			daemon.Stop()
		}
	})
}

func (daemon *Daemon) registerSessionPID(request RenderRequest) {
	pid, ok := parseSessionPID(request.SessionID)
	if !ok {
		return
	}

	var shellName string
	if request.Flags != nil {
		shellName = request.Flags.Shell
	}

	daemon.sessions.Register(pid, "", shellName)

	daemon.mu.Lock()
	daemon.cancelIdleStopLocked()
	daemon.mu.Unlock()
}

func (daemon *Daemon) onSessionUnregister(pid int) {
	daemon.service.CompleteSession(strconv.Itoa(pid))
}

func (daemon *Daemon) onAllSessionsEnded() {
	daemon.mu.Lock()
	daemon.scheduleIdleStopLocked()
	daemon.mu.Unlock()
}

func parseSessionPID(sessionID string) (int, bool) {
	pid, err := strconv.Atoi(sessionID)
	if err != nil {
		return 0, false
	}

	if pid <= 0 {
		return 0, false
	}

	return pid, true
}
