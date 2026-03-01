package daemon

import "sync"

// PromptSessionStore tracks per-session update hubs and coordinates cleanup
// of per-session prompt engine state in the registry.
type PromptSessionStore struct {
	registry *EngineRegistry
	hubs     map[string]*SessionUpdateHub
	mu       sync.Mutex
}

func NewPromptSessionStore(registry *EngineRegistry) *PromptSessionStore {
	return &PromptSessionStore{
		registry: registry,
		hubs:     make(map[string]*SessionUpdateHub),
	}
}

func (store *PromptSessionStore) Hub(sessionID string) *SessionUpdateHub {
	store.mu.Lock()
	defer store.mu.Unlock()

	hub, ok := store.hubs[sessionID]
	if ok {
		return hub
	}

	hub = NewSessionUpdateHub()
	store.hubs[sessionID] = hub
	return hub
}

func (store *PromptSessionStore) RemoveSession(sessionID string) {
	store.mu.Lock()
	delete(store.hubs, sessionID)
	store.mu.Unlock()

	if store.registry == nil {
		return
	}

	store.registry.RemoveSession(sessionID)
}

func (store *PromptSessionStore) Count() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return len(store.hubs)
}
