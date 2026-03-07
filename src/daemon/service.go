package daemon

import (
	"context"
	"sync"

	runtimePkg "github.com/po1o/prompto/src/runtime"
)

type RenderRequest struct {
	Flags     *runtimePkg.Flags
	SessionID string
	Repaint   bool
}

const renderCompletePayload = "__prompto_render_complete__"

type RenderResponse struct {
	Bundle   PromptBundle
	Type     string
	Segment  string
	Sequence uint64
}

type Service struct {
	// runtime owns request gating and per-session hubs/engines.
	runtime *SessionRenderRuntime
	// pipeline executes actual prompt rendering strategy (full render vs repaint).
	pipeline *RenderPipeline
	// renders keeps the currently active render stream handle by session.
	renders map[string]*ActiveRender
	mu      sync.Mutex
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
	var staleCompleted *ActiveRender
	var previous *ActiveRender
	if ok && existing != nil && service.isCompletedRender(existing) {
		delete(service.renders, request.SessionID)
		staleCompleted = existing
		existing = nil
		ok = false
	}

	if ok && existing != nil && !request.Repaint {
		// Non-repaint starts a new render generation; cancel the previous one.
		delete(service.renders, request.SessionID)
		previous = existing
	}
	service.mu.Unlock()

	if staleCompleted != nil {
		staleCompleted.Complete()
	}
	if previous != nil {
		previous.Complete()
	}

	bundle, active := service.pipeline.Start(request.SessionID, request.Flags, request.Repaint)
	sequence := service.currentSequence(request.SessionID)

	service.mu.Lock()
	if active == nil {
		delete(service.renders, request.SessionID)
	} else {
		service.renders[request.SessionID] = active
	}
	service.mu.Unlock()

	return RenderResponse{
		Type:     "initial",
		Bundle:   bundle,
		Sequence: sequence,
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
		if ctx != nil && ctx.Err() != nil {
			return RenderResponse{}, false
		}

		service.releaseActiveRenderIfCurrent(sessionID, active)
		return RenderResponse{}, false
	}

	if update.Snapshot.Payload == renderCompletePayload {
		service.releaseActiveRenderIfCurrent(sessionID, active)
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
		// Ensure request gate "active" counter is released.
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

func (service *Service) releaseActiveRenderIfCurrent(sessionID string, expected *ActiveRender) {
	if expected == nil {
		return
	}

	service.mu.Lock()
	current, ok := service.renders[sessionID]
	if !ok || current != expected {
		service.mu.Unlock()
		return
	}

	delete(service.renders, sessionID)
	service.mu.Unlock()

	expected.Complete()
}

func (service *Service) currentSequence(sessionID string) uint64 {
	if service.runtime == nil {
		return 0
	}

	hub := service.runtime.SessionHub(sessionID)
	if hub == nil {
		return 0
	}

	snapshot, ok := hub.Last()
	if !ok {
		return 0
	}

	return snapshot.Sequence
}

func (service *Service) isCompletedRender(active *ActiveRender) bool {
	if active == nil || active.handle == nil || active.handle.Hub() == nil {
		return false
	}

	snapshot, ok := active.handle.Hub().Last()
	if !ok {
		return false
	}

	if snapshot.Payload != renderCompletePayload {
		return false
	}

	renderID := active.handle.RenderID()
	if snapshot.RenderID == 0 || snapshot.RenderID == renderID {
		return true
	}

	return false
}
