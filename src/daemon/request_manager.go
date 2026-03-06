package daemon

import (
	"sync"

	"github.com/po1o/prompto/src/runtime"
)

type RequestHandle struct {
	// Render is the active generation metadata for one session request.
	Render *RenderHandle
	// releaseActive decrements ReloadGate active counter.
	releaseActive func()
	// once guarantees gate release and cancellation happen exactly once.
	once sync.Once
}

func (h *RequestHandle) Complete() {
	if h == nil {
		return
	}

	h.once.Do(func() {
		if h.Render != nil {
			h.Render.Complete()
		}

		if h.releaseActive != nil {
			h.releaseActive()
		}
	})
}

type RequestManager struct {
	// gate blocks new requests during reload and waits for active requests.
	gate *ReloadGate
	// coordinator handles session engine reuse + cancel/reattach behavior.
	coordinator *RenderCoordinator
}

func NewRequestManager(registry *EngineRegistry, gate *ReloadGate) *RequestManager {
	if gate == nil {
		gate = NewReloadGate()
	}

	return &RequestManager{
		gate:        gate,
		coordinator: NewRenderCoordinator(registry),
	}
}

func (manager *RequestManager) StartRequest(sessionID string, flags *runtime.Flags, repaint bool) *RequestHandle {
	release := manager.gate.StartRequest()
	render := manager.coordinator.StartRender(sessionID, flags, repaint)
	return &RequestHandle{
		Render:        render,
		releaseActive: release,
	}
}

func (manager *RequestManager) Reload(action func()) {
	manager.gate.BeginReload()
	defer manager.gate.EndReload()

	if action == nil {
		return
	}

	action()
}

func (manager *RequestManager) Snapshot() (active int, reloading bool) {
	return manager.gate.Snapshot()
}
