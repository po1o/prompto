package daemon

import "context"

type StreamRelay struct {
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
		update := relay.hub.Subscribe(currentAfter)

		select {
		case snapshot, ok := <-update:
			if !ok {
				return UpdateSnapshot{}, false
			}

			if snapshot.Sequence > currentAfter {
				currentAfter = snapshot.Sequence
			}

			if renderID == 0 || snapshot.RenderID == 0 || snapshot.RenderID == renderID {
				return snapshot, true
			}
		case <-ctx.Done():
			return UpdateSnapshot{}, false
		}
	}
}
