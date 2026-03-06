package daemon

import "sync"

type UpdateSnapshot struct {
	Payload  string
	RenderID uint64
	Sequence uint64
}

type SessionUpdateHub struct {
	waiters  []updateWaiter
	updates  []UpdateSnapshot
	sequence uint64
	mu       sync.Mutex
}

type updateWaiter struct {
	ch    chan UpdateSnapshot
	after uint64
}

func NewSessionUpdateHub() *SessionUpdateHub {
	return &SessionUpdateHub{}
}

func (hub *SessionUpdateHub) Publish(payload string, renderID ...uint64) UpdateSnapshot {
	var id uint64
	if len(renderID) > 0 {
		id = renderID[0]
	}

	hub.mu.Lock()
	hub.sequence++
	snapshot := UpdateSnapshot{
		Sequence: hub.sequence,
		Payload:  payload,
		RenderID: id,
	}
	hub.updates = append(hub.updates, snapshot)
	ready := make([]chan UpdateSnapshot, 0, len(hub.waiters))
	pending := make([]updateWaiter, 0, len(hub.waiters))
	for _, waiter := range hub.waiters {
		if snapshot.Sequence > waiter.after {
			ready = append(ready, waiter.ch)
			continue
		}

		pending = append(pending, waiter)
	}
	hub.waiters = pending
	hub.mu.Unlock()

	for _, waiter := range ready {
		waiter <- snapshot
		close(waiter)
	}

	return snapshot
}

func (hub *SessionUpdateHub) Last() (UpdateSnapshot, bool) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	if hub.sequence == 0 {
		return UpdateSnapshot{}, false
	}

	return hub.updates[len(hub.updates)-1], true
}

func (hub *SessionUpdateHub) Subscribe(after uint64) <-chan UpdateSnapshot {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	channel := make(chan UpdateSnapshot, 1)
	if snapshot, ok := hub.nextAfterLocked(after); ok {
		channel <- snapshot
		close(channel)
		return channel
	}

	hub.waiters = append(hub.waiters, updateWaiter{
		after: after,
		ch:    channel,
	})
	return channel
}

func (hub *SessionUpdateHub) nextAfterLocked(after uint64) (UpdateSnapshot, bool) {
	for _, snapshot := range hub.updates {
		if snapshot.Sequence <= after {
			continue
		}

		return snapshot, true
	}

	return UpdateSnapshot{}, false
}
