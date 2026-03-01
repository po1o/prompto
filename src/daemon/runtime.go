package daemon

import (
	"github.com/jandedobbeleer/oh-my-posh/src/prompt"
	runtimePkg "github.com/jandedobbeleer/oh-my-posh/src/runtime"
)

type SessionRenderHandle struct {
	request *RequestHandle
	relay   *StreamRelay
}

func (h *SessionRenderHandle) Engine() *prompt.Engine {
	if h == nil || h.request == nil || h.request.Render == nil {
		return nil
	}

	return h.request.Render.Engine
}

func (h *SessionRenderHandle) Relay() *StreamRelay {
	if h == nil {
		return nil
	}

	return h.relay
}

func (h *SessionRenderHandle) Complete() {
	engine := h.Engine()
	if engine != nil {
		ClearSegmentUpdates(engine)
	}

	if h.request != nil {
		h.request.Complete()
	}
}

type SessionRenderRuntime struct {
	requests *RequestManager
	sessions *PromptSessionStore
}

func NewSessionRenderRuntime(registry *EngineRegistry, gate *ReloadGate) *SessionRenderRuntime {
	return &SessionRenderRuntime{
		requests: NewRequestManager(registry, gate),
		sessions: NewPromptSessionStore(registry),
	}
}

func (sessionRuntime *SessionRenderRuntime) StartRequest(sessionID string, flags *runtimePkg.Flags, repaint bool) *SessionRenderHandle {
	request := sessionRuntime.requests.StartRequest(sessionID, flags, repaint)
	BindSegmentUpdates(sessionID, request.Render.Engine, sessionRuntime.sessions)
	hub := sessionRuntime.sessions.Hub(sessionID)
	return &SessionRenderHandle{
		request: request,
		relay:   NewStreamRelay(hub),
	}
}

func (sessionRuntime *SessionRenderRuntime) Reload(action func()) {
	sessionRuntime.requests.Reload(action)
}

func (sessionRuntime *SessionRenderRuntime) RemoveSession(sessionID string) {
	sessionRuntime.sessions.RemoveSession(sessionID)
}

func (sessionRuntime *SessionRenderRuntime) Snapshot() (active int, reloading bool) {
	return sessionRuntime.requests.Snapshot()
}

func (sessionRuntime *SessionRenderRuntime) SessionHub(sessionID string) *SessionUpdateHub {
	return sessionRuntime.sessions.Hub(sessionID)
}
