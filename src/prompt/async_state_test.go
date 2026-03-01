package prompt

import (
	"testing"
	"time"

	"github.com/jandedobbeleer/oh-my-posh/src/config"
	"github.com/stretchr/testify/require"
)

func TestSegmentStateTransitions(t *testing.T) {
	engine := &Engine{
		segmentStates: make(map[string]*segmentAsyncState),
	}

	segment := &config.Segment{
		Type:  config.TEXT,
		Alias: "text.main",
	}

	engine.prepareSegmentStates([]*config.Segment{segment}, false)
	state := engine.getSegmentState("text.main")
	require.Equal(t, stateExecute, state.State)
	require.True(t, state.RenderedAt.IsZero())

	engine.markSegmentPending(segment)
	state = engine.getSegmentState("text.main")
	require.Equal(t, statePending, state.State)

	engine.markSegmentDone(segment)
	state = engine.getSegmentState("text.main")
	require.Equal(t, stateDone, state.State)

	renderedAt := time.Now()
	engine.markSegmentRendered(segment, renderedAt)
	state = engine.getSegmentState("text.main")
	require.Equal(t, stateRendered, state.State)
	require.Equal(t, renderedAt, state.RenderedAt)
}

func TestPrepareSegmentStatesRepaintKeepsRenderedExceptVim(t *testing.T) {
	engine := &Engine{
		segmentStates: make(map[string]*segmentAsyncState),
	}

	renderedAt := time.Now().Add(-time.Minute)
	engine.setSegmentState("path.main", segmentAsyncState{State: stateRendered, RenderedAt: renderedAt})
	engine.setSegmentState("vim.main", segmentAsyncState{State: stateRendered, RenderedAt: renderedAt})

	path := &config.Segment{Type: config.PATH, Alias: "path.main"}
	vim := &config.Segment{Type: config.SegmentType("vim"), Alias: "vim.main"}
	engine.prepareSegmentStates([]*config.Segment{path, vim}, true)

	pathState := engine.getSegmentState("path.main")
	require.Equal(t, stateRendered, pathState.State)
	require.Equal(t, renderedAt, pathState.RenderedAt)

	vimState := engine.getSegmentState("vim.main")
	require.Equal(t, stateExecute, vimState.State)
	require.Equal(t, renderedAt, vimState.RenderedAt)
}
