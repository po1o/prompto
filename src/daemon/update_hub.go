package daemon

import "sync"

type UpdateSnapshot struct {
	Sequence uint64
	Payload  string
}

type SessionUpdateHub struct {
	mu       sync.Mutex
	sequence uint64
	last     string
	waiters  []chan UpdateSnapshot
}

func NewSessionUpdateHub() *SessionUpdateHub {
	return &SessionUpdateHub{}
}

func (hub *SessionUpdateHub) Publish(payload string) UpdateSnapshot {
	hub.mu.Lock()
	hub.sequence++
	snapshot := UpdateSnapshot{
		Sequence: hub.sequence,
		Payload:  payload,
	}
	hub.last = payload
	waiters := hub.waiters
	hub.waiters = nil
	hub.mu.Unlock()

	for _, waiter := range waiters {
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

	return UpdateSnapshot{
		Sequence: hub.sequence,
		Payload:  hub.last,
	}, true
}

func (hub *SessionUpdateHub) Subscribe(after uint64) <-chan UpdateSnapshot {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	channel := make(chan UpdateSnapshot, 1)
	if hub.sequence > after {
		channel <- UpdateSnapshot{
			Sequence: hub.sequence,
			Payload:  hub.last,
		}
		close(channel)
		return channel
	}

	hub.waiters = append(hub.waiters, channel)
	return channel
}
