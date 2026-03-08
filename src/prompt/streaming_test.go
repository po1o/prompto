package prompt

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/runtime"
	"github.com/po1o/prompto/src/segments/options"
	"github.com/po1o/prompto/src/shell"

	"github.com/stretchr/testify/require"
)

type slowWriter struct {
	text  string
	delay time.Duration
}

type countedSlowWriter struct {
	slowWriter
}

var countedSlowWriterExecutions atomic.Int32

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

func (w *countedSlowWriter) Enabled() bool {
	countedSlowWriterExecutions.Add(1)
	return w.slowWriter.Enabled()
}

func (w *countedSlowWriter) Init(properties options.Provider, env runtime.Environment) {
	w.slowWriter.Init(properties, env)
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
prompt:
  - segments: ["slow.main"]

slow.main:
  type: "slow_test"
  template: "SLOW"
  style: "plain"
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

func TestPrimaryStreamingLayoutReturnsPendingThenUpdates(t *testing.T) {
	segmentType := config.SegmentType("slow_test_layout")
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
  type: "slow_test_layout"
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

func TestPrimaryStreamingIncludesTransientPromptsInEveryRenderState(t *testing.T) {
	segmentType := config.SegmentType("slow_test_transient")
	previous, hadPrevious := config.Segments[segmentType]
	config.Segments[segmentType] = func() config.SegmentWriter { return &countedSlowWriter{} }
	t.Cleanup(func() {
		if hadPrevious {
			config.Segments[segmentType] = previous
			return
		}

		delete(config.Segments, segmentType)
	})

	countedSlowWriterExecutions.Store(0)

	configPath := filepath.Join(t.TempDir(), "slow-transient-streaming.omp.yaml")
	cfg := `
daemon_timeout: 50
render_pending_icon: "P:"
prompt:
  - segments: ["slow.main"]
rprompt:
  - segments: ["slow.right"]
transient:
  - segments: ["slow.transient"]
rtransient:
  - segments: ["slow.rtransient"]

slow.main:
  type: "slow_test_transient"
  template: "MAIN"
  style: "plain"

slow.right:
  type: "slow_test_transient"
  template: "RIGHT"
  style: "plain"

slow.transient:
  type: "slow_test_transient"
  template: "TRANSIENT"
  style: "plain"

slow.rtransient:
  type: "slow_test_transient"
  template: "RTRANSIENT"
  style: "plain"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	flags := &runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
	}
	engine := New(flags)

	updates := make(chan string, 16)
	initial := engine.PrimaryStreaming(context.Background(), 50*time.Millisecond, func(segment string) {
		updates <- segment
	})

	require.Contains(t, initial, "P:...")
	require.Contains(t, engine.StreamingRPrompt(), "P:...")
	require.Contains(t, engine.StreamingTransientPrompt(), "P:...")
	require.Contains(t, engine.StreamingTransientRPrompt(), "P:...")

	var seenComplete bool
	require.Eventually(t, func() bool {
		select {
		case update := <-updates:
			if update == "" {
				seenComplete = true
			}
		default:
		}

		return seenComplete && engine.PendingSegmentCount() == 0
	}, 2*time.Second, 20*time.Millisecond)

	require.Contains(t, engine.ReRender(), "MAIN")
	require.Contains(t, engine.StreamingRPrompt(), "RIGHT")
	require.Contains(t, engine.StreamingTransientPrompt(), "TRANSIENT")
	require.Contains(t, engine.StreamingTransientRPrompt(), "RTRANSIENT")
	require.Equal(t, int32(1), countedSlowWriterExecutions.Load())
}

func TestPrimaryRepaintLayoutReEvaluatesVimSegment(t *testing.T) {
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

func TestPrimaryRepaintSynchronizesStreamingStateAccess(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "repaint-sync.omp.yaml")
	cfg := `
prompt:
  - segments: ["session"]

session:
  type: "session"
  style: "plain"
  template: "L"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	flags := &runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
	}
	engine := New(flags)
	_ = engine.PrimaryStreaming(context.Background(), 50*time.Millisecond, func(string) {})

	const hold = 80 * time.Millisecond
	locked := make(chan struct{})
	go func() {
		engine.streamingMu.Lock()
		close(locked)
		time.Sleep(hold)
		engine.streamingMu.Unlock()
	}()
	<-locked

	start := time.Now()
	_ = engine.PrimaryRepaint()
	elapsed := time.Since(start)
	require.GreaterOrEqual(t, elapsed, hold-(10*time.Millisecond))
}

type repaintExecutionWriter struct{}

var repaintExecutionCount atomic.Int64
var repaintTemplateCount atomic.Int64
var repaintDangerCount atomic.Int64

func (w *repaintExecutionWriter) Enabled() bool {
	repaintExecutionCount.Add(1)
	return true
}
func (w *repaintExecutionWriter) Template() string {
	repaintTemplateCount.Add(1)
	return "X"
}
func (w *repaintExecutionWriter) SetText(string)                                 {}
func (w *repaintExecutionWriter) SetIndex(_ int)                                 {}
func (w *repaintExecutionWriter) Text() string                                   { return "X" }
func (w *repaintExecutionWriter) Init(_ options.Provider, _ runtime.Environment) {}
func (w *repaintExecutionWriter) CacheKey() (string, bool)                       { return "", false }

type repaintTemplateGuardWriter struct {
	text string
}

func (w *repaintTemplateGuardWriter) Enabled() bool                                  { return true }
func (w *repaintTemplateGuardWriter) Template() string                               { return "{{ .Text }}" }
func (w *repaintTemplateGuardWriter) SetText(text string)                            { w.text = text }
func (w *repaintTemplateGuardWriter) SetIndex(_ int)                                 {}
func (w *repaintTemplateGuardWriter) Text() string                                   { return w.text }
func (w *repaintTemplateGuardWriter) Init(_ options.Provider, _ runtime.Environment) {}
func (w *repaintTemplateGuardWriter) CacheKey() (string, bool)                       { return "", false }
func (w *repaintTemplateGuardWriter) Danger() bool {
	repaintDangerCount.Add(1)
	return true
}

func TestPrimaryRepaintDoesNotExecuteNonVimSegments(t *testing.T) {
	segmentType := config.SegmentType("repaint_execute_guard")
	previous, hadPrevious := config.Segments[segmentType]
	config.Segments[segmentType] = func() config.SegmentWriter {
		return &repaintExecutionWriter{}
	}
	t.Cleanup(func() {
		if hadPrevious {
			config.Segments[segmentType] = previous
			return
		}

		delete(config.Segments, segmentType)
	})
	repaintExecutionCount.Store(0)
	repaintTemplateCount.Store(0)

	configPath := filepath.Join(t.TempDir(), "repaint-non-exec.omp.yaml")
	cfg := `
prompt:
  - segments: ["test.main"]

test.main:
  type: "repaint_execute_guard"
  style: "plain"
  template: "X"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	flags := &runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
	}
	engine := New(flags)
	_ = engine.PrimaryRepaint()

	require.Equal(t, int64(0), repaintExecutionCount.Load())
}

func TestPrimaryRepaintDoesNotRenderNonVimSegmentsWithEmptyText(t *testing.T) {
	segmentType := config.SegmentType("repaint_render_guard")
	previous, hadPrevious := config.Segments[segmentType]
	config.Segments[segmentType] = func() config.SegmentWriter {
		return &repaintExecutionWriter{}
	}
	t.Cleanup(func() {
		if hadPrevious {
			config.Segments[segmentType] = previous
			return
		}

		delete(config.Segments, segmentType)
	})
	repaintExecutionCount.Store(0)
	repaintTemplateCount.Store(0)

	configPath := filepath.Join(t.TempDir(), "repaint-non-render.omp.yaml")
	cfg := `
prompt:
  - segments: ["test.main"]

test.main:
  type: "repaint_render_guard"
  style: "plain"
  template: "X"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	flags := &runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
	}
	engine := New(flags)
	engine.streamingBlocks = engine.resolveStreamingBlocks()

	require.NotEmpty(t, engine.streamingBlocks)
	segment := engine.streamingBlocks[0].Segments[0]
	require.NoError(t, segment.MapSegmentWithWriter(engine.Env))
	segment.Enabled = true
	segment.SetText("")
	repaintTemplateCount.Store(0)

	_ = engine.PrimaryRepaint()
	require.Equal(t, int64(0), repaintTemplateCount.Load())
}

func TestPrimaryRepaintDoesNotReevaluatePreviousSegmentTemplates(t *testing.T) {
	guardType := config.SegmentType("repaint_template_guard")
	guardPrevious, guardHadPrevious := config.Segments[guardType]
	config.Segments[guardType] = func() config.SegmentWriter {
		return &repaintTemplateGuardWriter{}
	}
	t.Cleanup(func() {
		if guardHadPrevious {
			config.Segments[guardType] = guardPrevious
			return
		}

		delete(config.Segments, guardType)
	})

	nextType := config.SegmentType("repaint_template_next")
	nextPrevious, nextHadPrevious := config.Segments[nextType]
	config.Segments[nextType] = func() config.SegmentWriter {
		return &repaintExecutionWriter{}
	}
	t.Cleanup(func() {
		if nextHadPrevious {
			config.Segments[nextType] = nextPrevious
			return
		}

		delete(config.Segments, nextType)
	})

	repaintDangerCount.Store(0)

	configPath := filepath.Join(t.TempDir(), "repaint-template-guard.omp.yaml")
	cfg := `
prompt:
  - segments: ["test.guard", "test.next"]

test.guard:
  type: "repaint_template_guard"
  style: "plain"
  template: "G"
  foreground_templates:
    - '{{ if .Danger }}red{{ end }}'

test.next:
  type: "repaint_template_next"
  style: "plain"
  template: "N"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	flags := &runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
	}
	engine := New(flags)
	engine.streamingBlocks = engine.resolveStreamingBlocks()

	require.Len(t, engine.streamingBlocks, 1)
	require.Len(t, engine.streamingBlocks[0].Segments, 2)

	guardSegment := engine.streamingBlocks[0].Segments[0]
	require.NoError(t, guardSegment.MapSegmentWithWriter(engine.Env))
	guardSegment.Enabled = true
	guardSegment.SetText("G")

	nextSegment := engine.streamingBlocks[0].Segments[1]
	require.NoError(t, nextSegment.MapSegmentWithWriter(engine.Env))
	nextSegment.Enabled = true
	nextSegment.SetText("N")

	_ = engine.PrimaryRepaint()
	require.Equal(t, int64(0), repaintDangerCount.Load())
}
