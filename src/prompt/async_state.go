package prompt

import (
	"time"

	"github.com/po1o/prompto/src/config"
)

type segmentExecutionState string

const (
	stateExecute  segmentExecutionState = "execute"
	statePending  segmentExecutionState = "pending"
	stateDone     segmentExecutionState = "done"
	stateRendered segmentExecutionState = "rendered"
)

type segmentAsyncState struct {
	RenderedAt time.Time
	State      segmentExecutionState
}

func (e *Engine) prepareSegmentStates(segments []*config.Segment, repaint bool) {
	for _, segment := range segments {
		key := segmentStateKey(segment)
		current := e.getSegmentState(key)

		if repaint {
			if segment.Type == config.SegmentType("vim") {
				e.setSegmentState(key, segmentAsyncState{State: stateExecute, RenderedAt: current.RenderedAt})
				continue
			}

			if current.State == "" {
				e.setSegmentState(key, segmentAsyncState{State: stateExecute})
			}

			continue
		}

		e.setSegmentState(key, segmentAsyncState{State: stateExecute})
	}
}

func (e *Engine) markSegmentPending(segment *config.Segment) {
	key := segmentStateKey(segment)
	current := e.getSegmentState(key)
	e.setSegmentState(key, segmentAsyncState{State: statePending, RenderedAt: current.RenderedAt})
}

func (e *Engine) markSegmentDone(segment *config.Segment) {
	key := segmentStateKey(segment)
	current := e.getSegmentState(key)
	e.setSegmentState(key, segmentAsyncState{State: stateDone, RenderedAt: current.RenderedAt})
}

func (e *Engine) markSegmentRendered(segment *config.Segment, renderedAt time.Time) {
	key := segmentStateKey(segment)
	e.setSegmentState(key, segmentAsyncState{State: stateRendered, RenderedAt: renderedAt})
}

func (e *Engine) getSegmentState(key string) segmentAsyncState {
	e.stateMu.Lock()
	defer e.stateMu.Unlock()

	state, ok := e.segmentStates[key]
	if !ok {
		return segmentAsyncState{}
	}

	return *state
}

func (e *Engine) setSegmentState(key string, state segmentAsyncState) {
	e.stateMu.Lock()
	defer e.stateMu.Unlock()

	if e.segmentStates == nil {
		e.segmentStates = make(map[string]*segmentAsyncState)
	}

	updated := state
	e.segmentStates[key] = &updated
}

func segmentStateKey(segment *config.Segment) string {
	return segment.Name()
}
