package daemon

import "context"

type StreamRelay struct {
	hub *SessionUpdateHub
}

func NewStreamRelay(hub *SessionUpdateHub) *StreamRelay {
	return &StreamRelay{hub: hub}
}

func (relay *StreamRelay) Next(ctx context.Context, after uint64) (UpdateSnapshot, bool) {
	if relay == nil || relay.hub == nil {
		return UpdateSnapshot{}, false
	}

	update := relay.hub.Subscribe(after)

	select {
	case snapshot, ok := <-update:
		if !ok {
			return UpdateSnapshot{}, false
		}
		return snapshot, true
	case <-ctx.Done():
		return UpdateSnapshot{}, false
	}
}
