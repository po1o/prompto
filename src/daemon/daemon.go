package daemon

import (
	"context"
	"sync/atomic"

	"github.com/jandedobbeleer/oh-my-posh/src/prompt"
)

type Daemon struct {
	service *Service
	stopped atomic.Bool
}

func New(renderer promptBundleRenderer) *Daemon {
	registry := NewEngineRegistry(prompt.New)
	gate := NewReloadGate()
	return &Daemon{
		service: NewService(registry, gate, renderer),
	}
}

func (daemon *Daemon) StartRender(request RenderRequest) RenderResponse {
	if daemon.stopped.Load() {
		return RenderResponse{Type: "stopped"}
	}

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
	daemon.stopped.Store(true)
}
