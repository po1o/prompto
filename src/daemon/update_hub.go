package daemon

import (
	"slices"
	"sync"
)

type UpdateSnapshot struct {
	Payload  string
	RenderID uint64
	Sequence uint64
}

// historyLimit caps the catch-up window a session keeps. A render publishes one
// update per pending segment plus a completion, so this holds several whole
// generations. Without a cap the history grows for the life of the shell.
const historyLimit = 64

type SessionUpdateHub struct {
	// waiters are blocked subscribers waiting for sequence > after.
	waiters []updateWaiter
	// updates keeps the most recent history, oldest first, so late subscribers
	// can catch up without polling. Publish is the only writer and appends
	// after incrementing, so the entries are contiguous and end at sequence.
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
	// Trim in batches so the copy is amortised. Cloning rather than reslicing
	// releases the discarded prefix instead of leaving it pinned by the array.
	if len(hub.updates) > 2*historyLimit {
		hub.updates = slices.Clone(hub.updates[len(hub.updates)-historyLimit:])
	}
	// Split waiters into ready vs still-pending under lock, then notify outside lock.
	// This avoids sending on channels while holding the mutex.
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

// Sequence returns the highest sequence published so far, or zero when the hub
// is empty. A subscriber that reads it before a render generation starts is
// guaranteed to see every update that generation goes on to publish.
func (hub *SessionUpdateHub) Sequence() uint64 {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	return hub.sequence
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

// nextAfterLocked returns the first retained update newer than after. The
// entries are contiguous and end at hub.sequence, so the successor's position
// is arithmetic rather than a search. A subscriber older than the retained
// window resumes at the oldest entry held: every update carries a whole prompt,
// so it converges on the current one without replaying what it skipped.
func (hub *SessionUpdateHub) nextAfterLocked(after uint64) (UpdateSnapshot, bool) {
	if after >= hub.sequence {
		return UpdateSnapshot{}, false
	}

	return hub.updates[max(len(hub.updates)-int(hub.sequence-after), 0)], true
}
