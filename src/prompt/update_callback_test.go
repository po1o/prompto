package prompt

import (
	"sync"
	"testing"

	"github.com/jandedobbeleer/oh-my-posh/src/config"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime"
	"github.com/jandedobbeleer/oh-my-posh/src/shell"

	"github.com/stretchr/testify/require"
)

func TestWriteBlockSegmentsInvokesUpdateCallbackPerSegment(t *testing.T) {
	engine := New(&runtime.Flags{
		Shell:         shell.GENERIC,
		TerminalWidth: 80,
	})

	var mu sync.Mutex
	updates := make(map[string]int)
	engine.SetUpdateCallback(func(segmentName string) {
		mu.Lock()
		defer mu.Unlock()
		updates[segmentName]++
	})

	block := &config.Block{
		Segments: []*config.Segment{
			{Type: config.TEXT, Alias: "text_a", Template: "A"},
			{Type: config.TEXT, Alias: "text_b", Template: "B"},
		},
	}

	_, _ = engine.writeBlockSegments(block)

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, updates["text_a"])
	require.Equal(t, 1, updates["text_b"])
	require.Len(t, updates, 2)
}
