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
	"github.com/po1o/prompto/src/terminal"

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

type blockingInitWriter struct {
	text string
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

func (w *countedSlowWriter) Enabled() bool {
	countedSlowWriterExecutions.Add(1)
	return w.slowWriter.Enabled()
}

func (w *countedSlowWriter) Init(properties options.Provider, env runtime.Environment) {
	w.slowWriter.Init(properties, env)
}

func (w *blockingInitWriter) Enabled() bool {
	time.Sleep(120 * time.Millisecond)
	return true
}

func (w *blockingInitWriter) Template() string {
	return "BLOCK"
}

func (w *blockingInitWriter) SetText(text string) {
	w.text = text
}

func (w *blockingInitWriter) SetIndex(_ int) {}

func (w *blockingInitWriter) Text() string {
	return w.text
}

func (w *blockingInitWriter) Init(_ options.Provider, _ runtime.Environment) {
	time.Sleep(25 * time.Millisecond)
}

func (w *blockingInitWriter) CacheKey() (string, bool) {
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
prompt:
  - segments: ["slow.main"]

slow.main:
  type: "slow_test"
  template: "SLOW"
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
	t.Cleanup(engine.WaitForSegmentExecutions)

	updates := make(chan string, 8)
	start := time.Now()
	initial, _ := engine.PrimaryStreaming(context.Background(), 50*time.Millisecond, func(segment string) {
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
  template: "SLOW"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	flags := &runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
	}
	engine := New(flags)
	t.Cleanup(engine.WaitForSegmentExecutions)

	updates := make(chan string, 8)
	start := time.Now()
	_, _ = engine.PrimaryStreaming(context.Background(), 50*time.Millisecond, func(segment string) {
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

slow.right:
  type: "slow_test_transient"
  template: "RIGHT"

slow.transient:
  type: "slow_test_transient"
  template: "TRANSIENT"

slow.rtransient:
  type: "slow_test_transient"
  template: "RTRANSIENT"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	flags := &runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
	}
	engine := New(flags)
	t.Cleanup(engine.WaitForSegmentExecutions)

	updates := make(chan string, 16)
	initial, _ := engine.PrimaryStreaming(context.Background(), 50*time.Millisecond, func(segment string) {
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

func TestPrimaryStreamingPendingRTransientPreservesBlockLeadingStyle(t *testing.T) {
	segmentType := config.SegmentType("slow_test_rtransient_rounded")
	previous, hadPrevious := config.Segments[segmentType]
	config.Segments[segmentType] = func() config.SegmentWriter { return &slowWriter{} }
	t.Cleanup(func() {
		if hadPrevious {
			config.Segments[segmentType] = previous
			return
		}

		delete(config.Segments, segmentType)
	})

	configPath := filepath.Join(t.TempDir(), "slow-rtransient-rounded.omp.yaml")
	cfg := `
daemon_timeout: 50
render_pending_icon: "P:"
rtransient:
  - leading_style: rounded
    trailing_style: rounded
    segments: ["slow.git", "text.time"]

slow.git:
  type: "slow_test_rtransient_rounded"
  style: "powerline"
  template: " GIT "
  foreground: "#ffffff"
  background: "#000000"

text.time:
  type: "text"
  style: "powerline"
  template: " TIME "
  foreground: "#ffffff"
  background: "#7a7a7a"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	flags := &runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
	}
	engine := New(flags)
	t.Cleanup(engine.WaitForSegmentExecutions)
	_, _ = engine.PrimaryStreaming(context.Background(), 50*time.Millisecond, func(string) {})

	pending := engine.StreamingTransientRPrompt()
	require.NotEmpty(t, pending)
	require.True(t, strings.HasPrefix(pending, "\uE0B6"), pending)
	require.Contains(t, pending, "P:...")
}

func TestPrimaryStreamingAndRepaintCanOverlap(t *testing.T) {
	segmentType := config.SegmentType("blocking_init_test")
	previous, hadPrevious := config.Segments[segmentType]
	config.Segments[segmentType] = func() config.SegmentWriter { return &blockingInitWriter{} }
	t.Cleanup(func() {
		if hadPrevious {
			config.Segments[segmentType] = previous
			return
		}

		delete(config.Segments, segmentType)
	})

	configPath := filepath.Join(t.TempDir(), "blocking-streaming.omp.yaml")
	cfg := `
daemon_timeout: 20
prompt:
  - segments: ["blocking.main"]

blocking.main:
  type: "blocking_init_test"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	flags := &runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
	}
	engine := New(flags)
	t.Cleanup(engine.WaitForSegmentExecutions)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = engine.PrimaryStreaming(context.Background(), 20*time.Millisecond, func(string) {})
	}()

	require.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			_ = engine.PrimaryRepaint()
			return false
		}
	}, 2*time.Second, 10*time.Millisecond)
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
  template: "L"

vim:
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
	t.Cleanup(engine.WaitForSegmentExecutions)

	_, _ = engine.PrimaryStreaming(context.Background(), 50*time.Millisecond, func(string) {})
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
  template: "L"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	flags := &runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
	}
	engine := New(flags)
	t.Cleanup(engine.WaitForSegmentExecutions)
	_, _ = engine.PrimaryStreaming(context.Background(), 50*time.Millisecond, func(string) {})

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
  template: "X"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	flags := &runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
	}
	engine := New(flags)
	t.Cleanup(engine.WaitForSegmentExecutions)
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
  template: "X"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	flags := &runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
	}
	engine := New(flags)
	t.Cleanup(engine.WaitForSegmentExecutions)
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
  template: "G"
  foreground_templates:
    - '{{ if .Danger }}red{{ end }}'

test.next:
  type: "repaint_template_next"
  template: "N"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	flags := &runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
	}
	engine := New(flags)
	t.Cleanup(engine.WaitForSegmentExecutions)
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

func newMergeTestEngine(t *testing.T) *Engine {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "merge-guard.omp.yaml")
	cfg := `
prompt:
  - segments: ["session"]

session:
  type: "session"
  template: "L"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	engine := New(&runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
	})
	t.Cleanup(engine.WaitForSegmentExecutions)

	return engine
}

// newMergePair returns a canonical segment and an executed worker clone of the
// given type, both with initialized writers, the clone marked Enabled as a
// merge marker.
func newMergePair(t *testing.T, engine *Engine, segmentType config.SegmentType) (canonical, executed *config.Segment) {
	t.Helper()

	canonical = &config.Segment{Type: segmentType}
	require.NoError(t, canonical.MapSegmentWithWriter(engine.Env))

	executed = canonical.Clone()
	require.NoError(t, executed.MapSegmentWithWriter(engine.Env))
	executed.Enabled = true

	return canonical, executed
}

func TestMergeStreamingResultSkippedAfterHardCancel(t *testing.T) {
	engine := newMergeTestEngine(t)
	canonical, executed := newMergePair(t, engine, config.SegmentType("text"))
	result := streamingResult{segment: canonical, executed: executed, key: "primary:0:0:Text"}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	engine.streamingMu.Lock()
	engine.mergeStreamingResultLocked(cancelled, result)
	engine.streamingMu.Unlock()
	require.False(t, canonical.Enabled, "hard-cancelled result must not merge into the canonical segment")

	engine.streamingMu.Lock()
	engine.mergeStreamingResultLocked(context.Background(), result)
	engine.streamingMu.Unlock()
	require.True(t, canonical.Enabled, "live context must merge the worker result")
}

func TestMergeStreamingResultSkipsVimAfterRepaint(t *testing.T) {
	engine := newMergeTestEngine(t)
	canonical, executed := newMergePair(t, engine, config.VIM)
	result := streamingResult{segment: canonical, executed: executed, key: "primary:0:0:Vim"}

	engine.streamingMu.Lock()
	engine.vimRepainted = true
	engine.mergeStreamingResultLocked(context.Background(), result)
	engine.streamingMu.Unlock()
	require.False(t, canonical.Enabled, "vim result executed before a repaint must not overwrite the repainted segment")

	engine.streamingMu.Lock()
	engine.vimRepainted = false
	engine.mergeStreamingResultLocked(context.Background(), result)
	engine.streamingMu.Unlock()
	require.True(t, canonical.Enabled, "cold-start vim result must merge when no repaint has occurred")
}

func TestPrimaryStreamingHardCancelStopsUpdatesAndPreservesPending(t *testing.T) {
	segmentType := config.SegmentType("slow_test_cancel")
	previous, hadPrevious := config.Segments[segmentType]
	config.Segments[segmentType] = func() config.SegmentWriter { return &slowWriter{} }
	t.Cleanup(func() {
		if hadPrevious {
			config.Segments[segmentType] = previous
			return
		}

		delete(config.Segments, segmentType)
	})

	configPath := filepath.Join(t.TempDir(), "slow-cancel.omp.yaml")
	cfg := `
daemon_timeout: 50
prompt:
  - segments: ["slow.main"]

slow.main:
  type: "slow_test_cancel"
  template: "SLOW"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	flags := &runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
	}
	engine := New(flags)
	t.Cleanup(engine.WaitForSegmentExecutions)

	ctx, cancel := context.WithCancel(context.Background())
	var updates atomic.Int32
	_, _ = engine.PrimaryStreaming(ctx, 50*time.Millisecond, func(string) {
		updates.Add(1)
	})
	require.NotEmpty(t, engine.PendingSegments())

	// Hard Cancel while the slow segment is still executing.
	cancel()
	engine.WaitForSegmentExecutions()

	require.Equal(t, int32(0), updates.Load(), "no updates may be published after a hard cancel")
	require.NotEmpty(t, engine.PendingSegments(), "a cancelled drain must not touch pending state")
}

// keep_when_empty exists so the prompt does not reflow when a segment has
// nothing to say. A repaint is exactly when that matters — a vim mode change
// redraws the line under the cursor — and the repaint path skips segments that
// did not render, so the kept one has to be let through explicitly.
func TestPrimaryRepaintKeepsSegmentsMarkedKeepWhenEmpty(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "keep-repaint.omp.yaml")
	cfg := `
prompt:
  - segments: ["session", "text.empty", "vim"]

session:
  type: "session"
  template: "L"

text.empty:
  type: "text"
  template: ""
  keep_when_empty: true
  leading_separator: "["
  trailing_separator: "]"
  foreground: "white"
  background: "blue"

vim:
  template: "{{ if .Insert }}INSERT{{ end }}{{ if .Normal }}NORMAL{{ end }}"
  foreground: "white"
  background: "green"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	flags := &runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
		VimMode:    "insert",
	}
	engine := New(flags)
	t.Cleanup(engine.WaitForSegmentExecutions)

	initial, _ := engine.PrimaryStreaming(context.Background(), 50*time.Millisecond, func(string) {})
	require.Contains(t, initial, "[]", "the kept segment should hold its shape on the first render")

	flags.VimMode = "normal"
	repainted := engine.PrimaryRepaint()

	require.Contains(t, repainted, "NORMAL", "expected the repaint to have happened")
	require.Contains(t, repainted, "[]", "the kept segment should still hold its shape after a repaint")
}

// The streaming blocks are built once and re-rendered for the life of the
// session, so the line's leading glyph — which writeSegment substitutes onto
// whichever segment happens to be first — must be taken back off again when the
// block ends. Left in place, a segment that is first in one render still
// carries the line's opening glyph in a later one where it is not, and draws it
// mid-line.
//
// This is why the restore moved from a defer in writeSegment to endBlock: the
// defer ran too early for the next segment, and never running it would leak the
// substitution into the next render.
func TestPrimaryStreamingRestoresTheBlockLeadingGlyph(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "leading-glyph.omp.yaml")
	cfg := `
prompt:
  - leading_style: rounded
    segments: ["text.first", "text.second"]

text.first:
  type: "text"
  template: " FIRST "
  foreground: "#ffffff"
  background: "#000000"

text.second:
  type: "text"
  template: " SECOND "
  foreground: "#ffffff"
  background: "#7a7a7a"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	flags := &runtime.Flags{
		ConfigPath: configPath,
		Plain:      true,
		Shell:      shell.GENERIC,
	}

	engine := New(flags)
	t.Cleanup(engine.WaitForSegmentExecutions)

	first, _ := engine.PrimaryStreaming(context.Background(), 50*time.Millisecond, func(string) {})
	require.NotEmpty(t, first)

	require.NotEmpty(t, engine.streamingBlocks)
	block := engine.streamingBlocks[0]
	require.Equal(t, "", block.LeadingGlyph, "the line should carry the compiled glyph")
	require.NotEmpty(t, block.Segments)

	for _, segment := range block.Segments {
		require.Empty(t, segment.LeadingGlyph,
			"%s kept the line's leading glyph after the block ended", segment.Name())
	}

	// The glyph still has to be drawn — the restore must not cost the opening.
	require.Contains(t, first, "")
}

// A segment colored only by a template must keep that color on every render
// after the first. Later renders draw it from the text it already produced and
// drop its templates, so the resolved colors have to be frozen onto the
// segment when it renders: a segment that configures neither `foreground` nor
// `background` would otherwise fall back to an empty color, losing its
// background and drawing the separator — which takes the block's color — in
// the terminal default.
func TestStreamingKeepsTemplatedColorsOnAnAlreadyRenderedSegment(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "templated-colors.omp.yaml")
	cfg := `
prompt:
  - segments: ["test.templated"]

test.templated:
  type: "text"
  template: "T"
  foreground_templates:
    - '{{ if true }}#00ff00{{ end }}'
  background_templates:
    - '{{ if true }}#ff0000{{ end }}'
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	plain := terminal.Plain
	t.Cleanup(func() {
		terminal.Plain = plain
	})

	engine := New(&runtime.Flags{
		ConfigPath:    configPath,
		Shell:         shell.GENERIC,
		TerminalWidth: 80,
	})
	t.Cleanup(engine.WaitForSegmentExecutions)

	const (
		onRedBackground   = "\x1b[48;2;255;0;0m"
		inGreenForeground = "\x1b[38;2;0;255;0m"
	)

	initial, _ := engine.PrimaryStreaming(context.Background(), 50*time.Millisecond, func(string) {})
	require.Contains(t, initial, onRedBackground)
	require.Contains(t, initial, inGreenForeground)

	// The update the daemon streams once a slow segment lands re-renders every
	// segment, including the ones already drawn.
	rendered := engine.ReRender()

	require.Contains(t, rendered, onRedBackground, "the templated background must survive the re-render")
	require.Contains(t, rendered, inGreenForeground, "the templated foreground must survive the re-render")
}

// slowerWriter finishes well after slowWriter, so a prompt holding one of each
// streams its two updates in a known order.
type slowerWriter struct {
	slowWriter
}

func (w *slowerWriter) Init(_ options.Provider, _ runtime.Environment) {
	w.delay = 450 * time.Millisecond
}

// The daemon re-renders the whole prompt on every streamed segment update, so
// a prompt with two slow segments draws the first one again when the second
// lands. That second frame is where a segment colored only by a template lost
// its color: the render reuses the text the segment already produced and drops
// its templates, leaving nothing behind to draw with.
//
// This is the sequence a detached HEAD produced — `git status` ran past the
// cutoff, so both the git segment and its transient twin streamed in, and the
// git segment went transparent the moment the second one arrived.
func TestStreamingUpdatesKeepTemplatedColorsInEveryFrame(t *testing.T) {
	fastType := config.SegmentType("slow_templated_fast")
	fastPrevious, fastHadPrevious := config.Segments[fastType]
	config.Segments[fastType] = func() config.SegmentWriter { return &slowWriter{} }
	t.Cleanup(func() {
		if fastHadPrevious {
			config.Segments[fastType] = fastPrevious
			return
		}

		delete(config.Segments, fastType)
	})

	slowType := config.SegmentType("slow_templated_slow")
	slowPrevious, slowHadPrevious := config.Segments[slowType]
	config.Segments[slowType] = func() config.SegmentWriter { return &slowerWriter{} }
	t.Cleanup(func() {
		if slowHadPrevious {
			config.Segments[slowType] = slowPrevious
			return
		}

		delete(config.Segments, slowType)
	})

	configPath := filepath.Join(t.TempDir(), "streamed-templated-colors.omp.yaml")
	cfg := `
prompt:
  - segments: ["streamed.templated", "streamed.plain"]

streamed.templated:
  type: "slow_templated_fast"
  template: "TEMPLATED"
  foreground: "#ffffff"
  background_templates:
    - '{{ if true }}#ff0000{{ end }}'

streamed.plain:
  type: "slow_templated_slow"
  template: "PLAIN"
  foreground: "#ffffff"
  background: "#0000ff"
`
	require.NoError(t, os.WriteFile(configPath, []byte(cfg), 0o644))

	plain := terminal.Plain
	t.Cleanup(func() {
		terminal.Plain = plain
	})

	engine := New(&runtime.Flags{
		ConfigPath:    configPath,
		Shell:         shell.GENERIC,
		TerminalWidth: 80,
	})
	t.Cleanup(engine.WaitForSegmentExecutions)

	const onRedBackground = "\x1b[48;2;255;0;0m"

	// Re-render on every update the way the daemon's render pipeline does.
	frames := make(chan string, 16)
	initial, _ := engine.PrimaryStreaming(context.Background(), 20*time.Millisecond, func(string) {
		frames <- engine.ReRender()
	})
	require.NotContains(t, initial, "TEMPLATED", "both segments should still be pending")

	var collected []string
	require.Eventually(t, func() bool {
		select {
		case frame := <-frames:
			collected = append(collected, frame)
		default:
		}

		for _, frame := range collected {
			if strings.Contains(frame, "TEMPLATED") && strings.Contains(frame, "PLAIN") {
				return true
			}
		}

		return false
	}, 3*time.Second, 20*time.Millisecond, "never saw a frame with both segments rendered")

	var sawReuse bool
	for _, frame := range collected {
		if !strings.Contains(frame, "TEMPLATED") {
			continue
		}

		require.Contains(t, frame, onRedBackground, "a streamed frame dropped the templated background")

		if strings.Contains(frame, "PLAIN") {
			sawReuse = true
		}
	}

	require.True(t, sawReuse, "the frame that re-draws an already-rendered segment is the one under test")
}
