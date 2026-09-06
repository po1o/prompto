package daemon

import "context"

type StreamRelay struct {
	// hub holds the recent update history for one session; it is bounded, so
	// a subscriber far enough behind resumes at the oldest entry still held.
	hub *SessionUpdateHub
}

func NewStreamRelay(hub *SessionUpdateHub) *StreamRelay {
	return &StreamRelay{hub: hub}
}

func (relay *StreamRelay) Next(ctx context.Context, after, renderID uint64) (UpdateSnapshot, bool) {
	if relay == nil || relay.hub == nil {
		return UpdateSnapshot{}, false
	}

	currentAfter := after

	for {
		// Subscribe(after) returns immediately if history already has a newer event.
		update := relay.hub.Subscribe(currentAfter)

		select {
		case snapshot, ok := <-update:
			if !ok {
				return UpdateSnapshot{}, false
			}

			if snapshot.Sequence > currentAfter {
				currentAfter = snapshot.Sequence
			}

			// renderID filter prevents old canceled render generations from leaking into stream.
			if renderID == 0 || snapshot.RenderID == 0 || snapshot.RenderID == renderID {
				return snapshot, true
			}
		case <-ctx.Done():
			return UpdateSnapshot{}, false
		}
	}
}
