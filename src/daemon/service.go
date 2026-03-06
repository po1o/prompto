package daemon

import (
	"context"
	"sync"

	runtimePkg "github.com/jandedobbeleer/oh-my-posh/src/runtime"
)

type RenderRequest struct {
	Flags     *runtimePkg.Flags
	SessionID string
	Repaint   bool
}

const renderCompletePayload = "__omp_render_complete__"

type RenderResponse struct {
	Bundle   PromptBundle
	Type     string
	Segment  string
	Sequence uint64
}

type Service struct {
	runtime  *SessionRenderRuntime
	pipeline *RenderPipeline
	renders  map[string]*ActiveRender
	mu       sync.Mutex
}

func NewService(registry *EngineRegistry, gate *ReloadGate, renderer promptBundleRenderer) *Service {
	sessionRuntime := NewSessionRenderRuntime(registry, gate)
	return &Service{
		runtime:  sessionRuntime,
		pipeline: NewRenderPipeline(sessionRuntime, renderer, nil),
		renders:  make(map[string]*ActiveRender),
	}
}

func (service *Service) StartRender(request RenderRequest) RenderResponse {
	service.mu.Lock()
	existing, ok := service.renders[request.SessionID]
	if ok && existing != nil && !request.Repaint {
		existing.Complete()
	}
	service.mu.Unlock()

	bundle, active := service.pipeline.Start(request.SessionID, request.Flags, request.Repaint)

	service.mu.Lock()
	service.renders[request.SessionID] = active
	service.mu.Unlock()

	return RenderResponse{
		Type:   "initial",
		Bundle: bundle,
	}
}

func (service *Service) NextUpdate(ctx context.Context, sessionID string, after uint64) (RenderResponse, bool) {
	service.mu.Lock()
	active, ok := service.renders[sessionID]
	service.mu.Unlock()
	if !ok || active == nil {
		return RenderResponse{}, false
	}

	update, ok := active.Next(ctx, after)
	if !ok {
		return RenderResponse{}, false
	}

	return RenderResponse{
		Type:     "update",
		Sequence: update.Snapshot.Sequence,
		Segment:  update.Snapshot.Payload,
		Bundle:   update.Bundle,
	}, true
}

func (service *Service) CompleteSession(sessionID string) {
	service.mu.Lock()
	active, ok := service.renders[sessionID]
	if ok {
		delete(service.renders, sessionID)
	}
	service.mu.Unlock()

	if ok && active != nil {
		active.Complete()
	}

	service.runtime.RemoveSession(sessionID)
}

func (service *Service) Reload(action func()) {
	service.runtime.Reload(action)
}

func (service *Service) Snapshot() (active int, reloading bool) {
	return service.runtime.Snapshot()
}

func (service *Service) SessionCount() int {
	service.mu.Lock()
	defer service.mu.Unlock()
	return len(service.renders)
}

func (service *Service) SessionHub(sessionID string) *SessionUpdateHub {
	return service.runtime.SessionHub(sessionID)
}

func (service *Service) Reset() {
	service.mu.Lock()
	activeRenders := make([]*ActiveRender, 0, len(service.renders))
	for sessionID, active := range service.renders {
		activeRenders = append(activeRenders, active)
		delete(service.renders, sessionID)
	}
	service.mu.Unlock()

	for _, active := range activeRenders {
		if active == nil {
			continue
		}

		active.Complete()
	}

	if service.runtime == nil {
		return
	}

	service.runtime.Reset()
}
