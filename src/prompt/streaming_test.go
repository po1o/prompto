package prompt

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jandedobbeleer/oh-my-posh/src/config"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime"
	"github.com/jandedobbeleer/oh-my-posh/src/segments/options"
	"github.com/jandedobbeleer/oh-my-posh/src/shell"

	"github.com/stretchr/testify/require"
)

type slowWriter struct {
	text  string
	delay time.Duration
}

func (w *slowWriter) Enabled() bool {
	time.Sleep(w.delay)
	return true
}

func (w *slowWriter) Template() string {
	return "{{ .Text }}"
}

func (w *slowWriter) SetText(text string) {
	w.text = text
}

func (w *slowWriter) SetIndex(_ int) {}

func (w *slowWriter) Text() string {
	return w.text
}

func (w *slowWriter) Init(_ options.Provider, _ runtime.Environment) {
	w.delay = 220 * time.Millisecond
}

func (w *slowWriter) CacheKey() (string, bool) {
	return "", false
}

func TestPrimaryStreamingLongSegmentReturnsPendingThenUpdates(t *testing.T) {
	segmentType := config.SegmentType("slow_test")
	previous, hadPrevious := config.Segments[segmentType]
	config.Segments[segmentType] = func() config.SegmentWriter { return &slowWriter{} }
	t.Cleanup(func() {
		if hadPrevious {
			config.Segments[segmentType] = previous
			return
		}

		delete(config.Segments, segmentType)
	})

	configPath := filepath.Join(t.TempDir(), "slow-streaming.omp.yaml")
	cfg := `
daemon_timeout: 50
blocks:
  - type: prompt
    segments:
      - type: slow_test
        alias: slow.main
        template: SLOW
        style: plain
        foreground: "#ffffff"
        background: "#000000"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	flags := &runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
	}
	engine := New(flags)

	updates := make(chan string, 8)
	start := time.Now()
	initial := engine.PrimaryStreaming(context.Background(), 50*time.Millisecond, func(segment string) {
		updates <- segment
	})
	elapsed := time.Since(start)

	require.Less(t, elapsed, 180*time.Millisecond)
	require.NotNil(t, engine.PendingSegments())
	require.NotEmpty(t, engine.PendingSegments())

	var seenSegmentUpdate bool
	var seenComplete bool
	require.Eventually(t, func() bool {
		select {
		case update := <-updates:
			if update == "slow.main" {
				seenSegmentUpdate = true
			}
			if update == "" {
				seenComplete = true
			}
		default:
		}

		return seenSegmentUpdate && seenComplete
	}, 2*time.Second, 20*time.Millisecond)

	_ = initial
	_ = engine.ReRender()
	require.Empty(t, engine.PendingSegments())
}

func TestPrimaryStreamingCompiledLayoutReturnsPendingThenUpdates(t *testing.T) {
	segmentType := config.SegmentType("slow_test_compiled")
	previous, hadPrevious := config.Segments[segmentType]
	config.Segments[segmentType] = func() config.SegmentWriter { return &slowWriter{} }
	t.Cleanup(func() {
		if hadPrevious {
			config.Segments[segmentType] = previous
			return
		}

		delete(config.Segments, segmentType)
	})

	configPath := filepath.Join(t.TempDir(), "slow-streaming.omp.yaml")
	cfg := `
daemon_timeout: 50
prompt:
  - segments: ["slow.main"]

slow.main:
  type: "slow_test_compiled"
  style: "plain"
  template: "SLOW"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	flags := &runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
	}
	engine := New(flags)

	updates := make(chan string, 8)
	start := time.Now()
	_ = engine.PrimaryStreaming(context.Background(), 50*time.Millisecond, func(segment string) {
		updates <- segment
	})
	elapsed := time.Since(start)

	require.Less(t, elapsed, 180*time.Millisecond)
	require.NotEmpty(t, engine.PendingSegments())

	var seenSegmentUpdate bool
	var seenComplete bool
	require.Eventually(t, func() bool {
		select {
		case update := <-updates:
			if update == "slow.main" {
				seenSegmentUpdate = true
			}
			if update == "" {
				seenComplete = true
			}
		default:
		}

		return seenSegmentUpdate && seenComplete
	}, 2*time.Second, 20*time.Millisecond)
}

func TestPrimaryRepaintCompiledLayoutReEvaluatesVimSegment(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "vim-repaint.omp.yaml")
	cfg := `
prompt:
  - segments: ["session"]
rprompt:
  - segments: ["vim"]

session:
  type: "session"
  style: "plain"
  template: "L"

vim:
  style: "plain"
  template: "{{ if .Insert }} INSERT {{ end }}{{ if .Normal }} NORMAL {{ end }}"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	flags := &runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
		VimMode:    "insert",
	}
	engine := New(flags)

	_ = engine.PrimaryStreaming(context.Background(), 50*time.Millisecond, func(string) {})
	require.True(t, strings.Contains(engine.StreamingRPrompt(), "INSERT"), "expected initial render to include INSERT mode")

	flags.VimMode = "normal"
	_ = engine.PrimaryRepaint()
	require.True(t, strings.Contains(engine.StreamingRPrompt(), "NORMAL"), "expected repaint to include NORMAL mode")
}
