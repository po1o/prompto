package prompt

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/po1o/prompto/src/color"
	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/log"
	runjobs "github.com/po1o/prompto/src/runtime/jobs"
	"github.com/po1o/prompto/src/shell"
	"github.com/po1o/prompto/src/template"
	"github.com/po1o/prompto/src/terminal"
)

// SegmentsTimedOut names the update announcing that segments outran the render
// deadline and are now drawn as timed out. It is a label for subscribers, not a
// completion: the render stays open, and a segment that finishes later still
// replaces its marker.
const SegmentsTimedOut = "__prompto_segments_timed_out__"

const (
	streamScopePrimary    = "primary"
	streamScopeTransient  = "transient"
	streamScopeRTransient = "rtransient"
)

func segmentKey(scope string, blockIndex, segmentIndex int, segment *config.Segment) string {
	return fmt.Sprintf("%s:%d:%d:%s", scope, blockIndex, segmentIndex, segment.Name())
}

type streamingBlockSet struct {
	scope  string
	blocks []*config.Block
}

type streamingSegment struct {
	segment *config.Segment
	// working is the worker's private clone, taken under streamingMu in
	// prepareStreamingSegments. Workers must never read the canonical
	// segment: the render path rewrites it for pending placeholders.
	working *config.Segment
	key     string
}

type streamingResult struct {
	segment *config.Segment
	// executed is the worker's private clone holding the computation result.
	// It is nil when execution did not finish cleanly (segment timeout): the
	// execution goroutine may then still be mutating the clone, so it cannot
	// be merged and the canonical segment keeps its pre-execution state.
	executed *config.Segment
	key      string
}

// mergeStreamingResultLocked copies a worker's completed computation from its
// private clone into the canonical segment. The canonical segment is owned by
// streamingMu — callers must hold it. This keeps segment execution decoupled
// from render state: workers never touch the segment the render path reads.
func (e *Engine) mergeStreamingResultLocked(ctx context.Context, result streamingResult) {
	if result.executed == nil {
		return
	}

	// Hard Cancel safety: cancellation strictly happens-before the next
	// render's prepareStreamingSegments, so re-checking under streamingMu
	// guarantees a superseded render's worker cannot merge stale state into
	// canonical segments the new generation has already re-prepared.
	if ctx.Err() != nil {
		return
	}

	// Soft Cancel (vim toggle) safety: once PrimaryRepaint has re-executed
	// the canonical vim segment with the current VimMode, a late worker merge
	// would overwrite it with a clone executed under the old mode. The vim
	// segment is trivially cheap and re-executed on every repaint, so the
	// async result is safely dropped. Before the first repaint of a render
	// generation the merge proceeds: cold starts rely on it for the initial
	// vim state.
	if result.segment.Type == config.VIM && e.vimRepainted {
		return
	}

	if err := result.segment.CopyWriterStateFrom(result.executed); err != nil {
		log.Error(err)
		return
	}

	result.segment.Needs = result.executed.Needs
	result.segment.Duration = result.executed.Duration
	result.segment.NameLength = result.executed.NameLength
}

// WaitForSegmentExecutions blocks until all in-flight segment execution
// goroutines have finished, including ones abandoned by a segment timeout.
// Segment executions may outlive PrimaryStreaming's render window by design;
// use this to quiesce the engine (tests, daemon shutdown).
func (e *Engine) WaitForSegmentExecutions() {
	e.executionWG.Wait()
}

// PrimaryStreaming renders a prompt with a timeout cutoff and pending
// placeholders. placeholderAfter is how long to wait before returning with
// them; timedOutAfter is how long a segment may then stay pending before it is
// drawn as timed out, and zero leaves it pending indefinitely.
//
// streaming reports whether the returned prompt carries pending placeholders,
// and so whether updateCallback will run: it is latched under the same lock
// that rendered the prompt, so the two always describe the same instant. The
// caller must not re-derive it from PendingSegmentCount afterwards — the
// publisher goroutine below can drain the last result microseconds later,
// making the count read 0 for a prompt that still shows placeholders.
func (e *Engine) PrimaryStreaming(
	ctx context.Context,
	placeholderAfter, timedOutAfter time.Duration,
	updateCallback func(string),
) (initial string, streaming bool) {
	if placeholderAfter <= 0 {
		placeholderAfter = 100 * time.Millisecond
	}

	if ctx == nil {
		ctx = context.Background()
	}

	e.resetSharedProviders()

	e.streamingMu.Lock()
	e.pendingSegments = make(map[string]bool)
	e.timedOutSegments = make(map[string]bool)
	e.cachedValues = make(map[string]string)
	e.segmentCacheKeys = make(map[string]string)
	// New render generation: its own vim worker result must merge (cold
	// start); only a repaint issued after this point makes it stale.
	e.vimRepainted = false
	e.streamingBlocks = e.resolveStreamingBlocks()
	e.streamingTransient = e.resolveStreamingTransientBlocks()
	e.streamingRTransient = e.resolveStreamingRTransientBlocks()

	segmentsToExecute, completed := e.prepareStreamingSegments()
	e.streamingMu.Unlock()
	results := e.startStreamingExecutions(ctx, segmentsToExecute)

	// Armed before the placeholder wait rather than after it: timedOutAfter
	// counts from the start of the render, and that wait is part of the render.
	// Arming it afterwards would push the marker out by placeholderAfter, which
	// a large daemon_timeout can make the difference between the marker
	// arriving and the client having already stopped listening for it.
	var timedOut *time.Timer
	if timedOutAfter > 0 {
		timedOut = time.NewTimer(timedOutAfter)
	}

	e.collectStreamingResultsUntil(ctx, placeholderAfter, results, completed)

	e.streamingMu.Lock()
	for _, entry := range segmentsToExecute {
		if completed[entry.key] {
			continue
		}

		e.pendingSegments[entry.key] = true
	}
	initialPrompt := e.renderStreamingPrompt()
	hasPending := len(e.pendingSegments) > 0
	e.streamingMu.Unlock()

	if !hasPending {
		if timedOut != nil {
			timedOut.Stop()
		}

		return initialPrompt, false
	}

	go e.publishStreamingResults(ctx, results, updateCallback, timedOut)

	return initialPrompt, true
}

// publishStreamingResults merges each late segment result and announces it,
// then announces completion with the empty segment name. It is the only writer
// of updates once PrimaryStreaming has returned, so a caller told streaming is
// true can rely on the completion announcement arriving unless ctx is
// cancelled — in which case the render generation is being torn down anyway.
//
// deadline caps how long a segment may stay pending before it is drawn as
// timed out, so the shell is left with something that says so rather than the
// pending placeholders it drew at the start. It does not end the render:
// completing it would retire the generation, and retiring it cancels the
// context the segment is still executing under, killing the very work that
// would have answered. A segment that arrives late still replaces its own
// marker, and still warms the cache for the next prompt.
func (e *Engine) publishStreamingResults(
	ctx context.Context,
	results <-chan streamingResult,
	updateCallback func(string),
	timedOut *time.Timer,
) {
	var expired <-chan time.Time
	if timedOut != nil {
		expired = timedOut.C
		defer timedOut.Stop()
	}

	for {
		select {
		case result, ok := <-results:
			if !ok {
				if ctx.Err() == nil {
					updateCallback("")
				}

				return
			}

			e.streamingMu.Lock()
			// Re-check under the lock: a Hard Cancel may have landed while this
			// goroutine was blocked acquiring streamingMu, in which case
			// pendingSegments already belongs to the next render generation and
			// must not be touched or published.
			if ctx.Err() != nil {
				e.streamingMu.Unlock()
				return
			}

			e.mergeStreamingResultLocked(ctx, result)
			delete(e.pendingSegments, result.key)
			delete(e.timedOutSegments, result.key)
			e.streamingMu.Unlock()

			updateCallback(result.segment.Name())
		case <-expired:
			// The timer fires once, but clear the arm so it cannot be chosen
			// again over a result that is ready at the same moment.
			expired = nil

			if !e.markPendingSegmentsTimedOut(ctx) {
				// Everything landed in the meantime, or this generation is
				// gone; either way there is nothing to mark and nothing to say.
				continue
			}

			updateCallback(SegmentsTimedOut)
		case <-ctx.Done():
			return
		}
	}
}

// markPendingSegmentsTimedOut flags every segment still outstanding so the
// render draws it as timed out, and reports whether there was any.
//
// The context is re-checked here rather than before the call, because the wait
// for streamingMu is exactly where a hard cancel lands: the next generation
// takes the lock, resets both maps and refills pendingSegments with its own
// segments, and only then is this granted the lock. Marking at that point
// paints a prompt milliseconds old as timed out, and the markers clear only as
// each segment reports, so a slow one stays wrongly red for the whole render.
func (e *Engine) markPendingSegmentsTimedOut(ctx context.Context) bool {
	e.streamingMu.Lock()
	defer e.streamingMu.Unlock()

	if ctx.Err() != nil || len(e.pendingSegments) == 0 {
		return false
	}

	for key := range e.pendingSegments {
		e.timedOutSegments[key] = true
	}

	return true
}

func (e *Engine) prepareStreamingSegments() ([]streamingSegment, map[string]bool) {
	segmentsToExecute := make([]streamingSegment, 0, 32)
	completed := make(map[string]bool)

	for _, set := range e.streamingBlockSets() {
		for blockIndex, block := range set.blocks {
			for segmentIndex, segment := range block.Segments {
				key := segmentKey(set.scope, blockIndex, segmentIndex, segment)
				_ = segment.MapSegmentWithWriter(e.Env)
				cacheKey := segment.DaemonCacheKey()
				e.segmentCacheKeys[key] = cacheKey

				entry, found, explicit := e.getSegmentCache(segment)
				if found {
					if explicit {
						duration := segment.Cache.Duration
						if duration.IsEmpty() || duration.Seconds() <= 0 {
							e.applySegmentCacheEntry(segment, entry)
							completed[key] = true
							continue
						}

						age := time.Since(entry.RenderedAt)
						if age <= time.Duration(duration.Seconds())*time.Second {
							e.applySegmentCacheEntry(segment, entry)
							completed[key] = true
							continue
						}
					}

					e.cachedValues[key] = entry.Text
				}

				segmentsToExecute = append(segmentsToExecute, streamingSegment{
					segment: segment,
					working: segment.Clone(),
					key:     key,
				})
			}
		}
	}

	return segmentsToExecute, completed
}

func (e *Engine) startStreamingExecutions(
	ctx context.Context,
	segments []streamingSegment,
) <-chan streamingResult {
	results := make(chan streamingResult, len(segments))
	var wg sync.WaitGroup

	sources := make(map[config.SegmentType]*config.Segment)
	for _, entry := range segments {
		if _, ok := e.sharedProviderFactory[entry.segment.Type]; !ok {
			continue
		}

		if _, ok := sources[entry.segment.Type]; ok {
			continue
		}

		// Derive from the worker clone, not the canonical segment; this runs
		// unlocked while a concurrent repaint may rewrite the canonical.
		sources[entry.segment.Type] = entry.working.Clone()
	}

	for _, entry := range segments {
		wg.Add(1)
		e.executionWG.Add(1)
		go func(entry streamingSegment) {
			defer wg.Done()
			defer e.executionWG.Done()
			e.executeStreamingSegment(ctx, entry, sources, results)
		}(entry)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

func (e *Engine) executeStreamingSegment(
	ctx context.Context,
	entry streamingSegment,
	sources map[config.SegmentType]*config.Segment,
	results chan<- streamingResult,
) {
	if ctx.Err() != nil {
		return
	}

	segment := entry.segment

	// Execute on the private clone taken under streamingMu: the canonical
	// segment is owned by that lock (the render path reads and temporarily
	// rewrites it for pending placeholders), so workers must never touch it.
	// The result is merged back under the lock when the result is collected.
	working := entry.working

	if providerFactory, ok := e.sharedProviderFactory[segment.Type]; ok {
		if err := working.MapSegmentWithWriter(e.Env); err == nil {
			sharedProvider := e.getOrCreateSharedProvider(ctx, segment.Type, sources[segment.Type], providerFactory)
			if res, sharedErr := sharedProvider.Get(); sharedErr == nil {
				_ = working.CopyWriterStateFrom(res.Source)
			}
		}

		if ctx.Err() != nil {
			return
		}

		results <- streamingResult{segment: segment, executed: working, key: entry.key}
		return
	}

	completed, clean := e.executeSegmentWithContext(ctx, working)
	if !completed {
		// Cancelled: executeSegmentWithContext already marked the clone
		// abandoned before killing, so its still-running execution goroutine
		// cannot register a stale writer in the template cache.
		return
	}

	if ctx.Err() != nil {
		return
	}

	if !clean {
		// Timeout: the execution goroutine still owns the clone (already marked
		// abandoned inside executeSegmentWithContext). Drop it so the stale
		// writer state is never merged back into the canonical segment.
		working = nil
	}

	results <- streamingResult{segment: segment, executed: working, key: entry.key}
}

// executeSegmentWithContext runs the segment's execution with cancellation and
// timeout handling. completed reports whether the caller should emit a result;
// clean reports whether the execution goroutine finished (false on timeout,
// when it may still be running and mutating the segment).
func (e *Engine) executeSegmentWithContext(ctx context.Context, segment *config.Segment) (completed, clean bool) {
	done := make(chan struct{})
	gidChan := make(chan uint64, 1)

	// Tracked in executionWG: on timeout this goroutine outlives the worker
	// (and possibly the render); WaitForSegmentExecutions joins it.
	e.executionWG.Go(func() {
		gidChan <- runjobs.CurrentGID()
		e.executeWithoutLegacySegmentCache(segment)
		close(done)
	})

	gid := <-gidChan

	// A nil channel blocks forever, which is what "no timeout" means here.
	var expired <-chan time.Time
	if segment.Timeout > 0 {
		expired = time.After(time.Duration(segment.Timeout) * time.Millisecond)
	}

	// Mark abandoned before killing: KillGoroutineChildren unblocks the hung
	// Execute, so the goroutine's deferred template-cache registration can run
	// the instant we kill. The atomic store must happen-before the kill for the
	// defer's isAbandoned() load to observe it and suppress the stale write.
	select {
	case <-done:
		return true, true
	case <-ctx.Done():
		segment.MarkAbandoned()
		_ = runjobs.KillGoroutineChildren(gid)
		return false, false
	case <-expired:
		log.Errorf("timeout after %dms for segment: %s", segment.Timeout, segment.Name())
		segment.MarkAbandoned()
		if err := runjobs.KillGoroutineChildren(gid); err != nil {
			log.Errorf("failed to kill child processes for goroutine %d (segment: %s): %v", gid, segment.Name(), err)
		}
		return true, false
	}
}

func (e *Engine) collectStreamingResultsUntil(
	ctx context.Context,
	timeout time.Duration,
	results <-chan streamingResult,
	completed map[string]bool,
) {
	cutoff := time.After(timeout)

	for {
		select {
		case result, ok := <-results:
			if !ok {
				return
			}

			e.streamingMu.Lock()
			e.mergeStreamingResultLocked(ctx, result)
			e.streamingMu.Unlock()
			completed[result.key] = true
		case <-ctx.Done():
			return
		case <-cutoff:
			return
		}
	}
}

// PrimaryRepaint re-renders prompt state for repaint without starting new computations.
// Only the vim segment is re-executed. Other segments are served from completed or pending cache state.
func (e *Engine) PrimaryRepaint() string {
	e.streamingMu.Lock()
	defer e.streamingMu.Unlock()
	e.repaintOnly = true
	defer func() {
		e.repaintOnly = false
	}()

	if e.pendingSegments == nil {
		e.pendingSegments = make(map[string]bool)
	}

	if e.timedOutSegments == nil {
		e.timedOutSegments = make(map[string]bool)
	}

	if e.cachedValues == nil {
		e.cachedValues = make(map[string]string)
	}

	if e.segmentCacheKeys == nil {
		e.segmentCacheKeys = make(map[string]string)
	}

	if len(e.streamingBlocks) == 0 {
		e.streamingBlocks = e.resolveStreamingBlocks()
	}

	for _, set := range e.streamingBlockSets() {
		for blockIndex, block := range set.blocks {
			for segmentIndex, segment := range block.Segments {
				key := segmentKey(set.scope, blockIndex, segmentIndex, segment)
				if segment.Type == config.VIM {
					_ = segment.MapSegmentWithWriter(e.Env)
					segment.Execute(e.Env)
					// The canonical vim segment now reflects the current
					// VimMode; block late async merges from overwriting it
					// with a clone executed under the previous mode.
					e.vimRepainted = true
					continue
				}

				// Repaint can occur without an active streaming render. Ensure writer is initialized
				// so rendering logic can safely inspect current text/cache state without execution.
				_ = segment.EnsureWriter(e.Env)

				cacheKey := e.segmentCacheKeys[key]
				if cacheKey == "" {
					cacheKey = segment.DaemonCacheKey()
					e.segmentCacheKeys[key] = cacheKey
				}

				if e.pendingSegments[key] {
					entry, found, _ := e.getSegmentCache(segment)
					if found {
						e.cachedValues[key] = entry.Text
					}
					continue
				}

				entry, found, _ := e.getSegmentCache(segment)
				if found {
					e.applySegmentCacheEntry(segment, entry)
				}
			}
		}
	}

	return e.renderStreamingPrompt()
}

// ReRender re-renders the prompt using current segment state.
func (e *Engine) ReRender() string {
	e.streamingMu.Lock()
	defer e.streamingMu.Unlock()
	return e.renderStreamingPrompt()
}

// StreamingRPrompt returns the right prompt from streaming render state.
func (e *Engine) StreamingRPrompt() string {
	return e.rprompt
}

func (e *Engine) renderStreamingPrompt() string {
	e.prompt.Reset()
	e.currentLineLength = 0
	e.rprompt = ""
	e.rpromptLength = 0

	needsPrimaryRightPrompt := e.needsPrimaryRightPrompt()
	if e.hasLayoutPrimary() {
		needsPrimaryRightPrompt = false
	}

	e.writePrimaryPromptStreaming(needsPrimaryRightPrompt)

	switch e.Env.Shell() {
	case shell.ZSH:
		if !e.Env.Flags().Eval {
			break
		}

		if e.isWarp() {
			e.writePrimaryRightPrompt()
			return fmt.Sprintf("PS1=%s", shell.QuotePosixStr(e.string()))
		}

		prompt := fmt.Sprintf("PS1=%s", shell.QuotePosixStr(e.string()))
		prompt += fmt.Sprintf("\nRPROMPT=%s", shell.QuotePosixStr(e.rprompt))
		return prompt
	default:
		if !needsPrimaryRightPrompt {
			break
		}

		e.writePrimaryRightPrompt()
	}

	return e.string()
}

func (e *Engine) writePrimaryPromptStreaming(needsPrimaryRPrompt bool) {
	if e.Config.ShellIntegration {
		exitCode, _ := e.Env.StatusCodes()
		e.write(terminal.CommandFinished(exitCode, e.Env.Flags().NoExitCode))
		e.write(terminal.PromptStart())
	}

	cycle = &e.Config.Cycle
	var cancelNewline, didRender bool

	blocks := e.streamingBlocks

	for i, block := range blocks {
		if i == 0 {
			row, _ := e.Env.CursorPosition()
			cancelNewline = e.Env.Flags().Cleared || e.Env.Flags().PromptCount == 1 || row == 1
		}

		if i != 0 {
			cancelNewline = !didRender
		}

		if block.Type == config.RPrompt && !needsPrimaryRPrompt && !e.hasLayoutPrimary() {
			continue
		}

		if e.renderBlockStreaming(block, i, cancelNewline) {
			didRender = true
		}
	}

	if len(e.Config.ConsoleTitleTemplate) > 0 && !e.Env.Flags().Plain {
		title := e.getTitleTemplateText()
		e.write(terminal.FormatTitle(title))
	}

	if e.Config.CursorPadding {
		e.write(" ")
		e.currentLineLength++
	}

	if e.Config.ITermFeatures != nil && e.isIterm() {
		host, _ := e.Env.Host()
		e.write(terminal.RenderItermFeatures(e.Config.ITermFeatures, e.Env.Shell(), e.Env.Pwd(), e.Env.User(), host))
	}

	if e.Config.ShellIntegration {
		e.write(terminal.CommandStart())
	}

	e.pwd()
}

func (e *Engine) resolveStreamingBlocks() []*config.Block {
	return e.layoutPrimaryBlocks()
}

func (e *Engine) renderBlockStreaming(block *config.Block, blockIndex int, cancelNewline bool) bool {
	blockText, length := e.writeBlockSegmentsStreaming(streamScopePrimary, block, blockIndex)
	return e.renderBlockWithText(block, blockText, length, cancelNewline)
}

func (e *Engine) writeBlockSegmentsStreaming(scope string, block *config.Block, blockIndex int) (string, int) {
	segmentIndex := 0

	saved := make([]segmentStyle, 0, len(block.Segments))

	for segmentPosition, segment := range block.Segments {
		key := segmentKey(scope, blockIndex, segmentPosition, segment)
		if e.pendingSegments[key] {
			saved = append(saved, styleOf(segment))
			e.writePlaceholderSegment(block, segment, key)
			continue
		}

		if segment.Text() != "" && segment.Enabled {
			saved = append(saved, styleOf(segment))

			segment.ForegroundTemplates = nil
			segment.BackgroundTemplates = nil
			segmentIndex++
			e.writeSegment(block, segment)
			continue
		}

		if e.repaintOnly && segment.Type != config.VIM {
			e.writeKeptSegment(block, segment)
			continue
		}

		if !segment.Render(segmentIndex, e.forceRender) {
			e.writeKeptSegment(block, segment)
			continue
		}

		segmentIndex++
		e.storeSegmentCache(segment, time.Now())
		bakeResolvedColors(segment)
		e.writeSegment(block, segment)
	}

	e.writeBlockSeparator(block)

	for i := range saved {
		saved[i].restore()
	}

	e.endBlock()

	return terminal.String()
}

// bakeResolvedColors freezes a segment's templated colors onto the segment
// itself, the way a restored cache entry does.
//
// Every later render of this generation draws the segment from the text it
// already produced, and drops its templates rather than evaluate them again
// against a writer whose state belongs to an earlier render. A segment colored
// only by a template configures neither Foreground nor Background, so without
// this it loses its color at that point: the background goes transparent, and
// the separator, which takes the block's color, falls back to the terminal
// default.
//
// Writing back is what ResolveForeground deliberately stopped doing, because
// making one render's answer the next render's fallback keeps a stale color on
// a segment whose template has since stopped matching. That cannot happen
// here: a block holds clones of the segment definitions, made fresh for each
// render generation, and within one generation a rendered segment is never
// executed again — so the template's inputs, and its answer, cannot change
// between the render that resolved the color and the renders that reuse it.
func bakeResolvedColors(segment *config.Segment) {
	foreground := segment.ResolveForeground()
	background := segment.ResolveBackground()

	segment.Foreground = foreground
	segment.Background = background
}

// segmentStyle is a segment's drawing state, taken before the render path
// rewrites it — for a pending placeholder, or to drop templates from a segment
// already rendered this generation — and put back once the block is written.
// Capture and restore live together so they cannot drift apart.
type segmentStyle struct {
	segment             *config.Segment
	text                string
	foreground          color.Ansi
	background          color.Ansi
	foregroundTemplates template.List
	backgroundTemplates template.List
	enabled             bool
}

func styleOf(segment *config.Segment) segmentStyle {
	return segmentStyle{
		segment:             segment,
		text:                segment.Text(),
		foreground:          segment.Foreground,
		background:          segment.Background,
		foregroundTemplates: segment.ForegroundTemplates,
		backgroundTemplates: segment.BackgroundTemplates,
		enabled:             segment.Enabled,
	}
}

func (style *segmentStyle) restore() {
	style.segment.SetText(style.text)
	style.segment.Foreground = style.foreground
	style.segment.Background = style.background
	style.segment.ForegroundTemplates = style.foregroundTemplates
	style.segment.BackgroundTemplates = style.backgroundTemplates
	style.segment.Enabled = style.enabled
}

// writePlaceholderSegment draws a segment that has not produced its value: the
// pending marker while it is still running, or the timeout marker once the
// render deadline passed with it still outstanding. The caller saves the
// segment's style first — this rewrites it.
func (e *Engine) writePlaceholderSegment(block *config.Block, segment *config.Segment, key string) {
	cached := e.cachedValues[key]

	// Both states drop the segment's own colour templates, so one coloured only
	// by those would draw transparent — and the separator, which takes the
	// block's colour, with it. The pending background backs both states.
	text, background := segment.GetPendingText(cached, e.Config)
	if e.timedOutSegments[key] {
		timedOutText, foreground := segment.GetTimeoutText(cached, e.Config)
		text = timedOutText
		segment.Foreground = foreground
	}

	if background == "" {
		background = "darkGray"
	}

	segment.SetText(text)
	segment.Background = background
	segment.ForegroundTemplates = nil
	segment.BackgroundTemplates = nil
	segment.Enabled = true

	e.writeSegment(block, segment)
}

func (e *Engine) streamingBlockSets() []streamingBlockSet {
	return []streamingBlockSet{
		{scope: streamScopePrimary, blocks: e.streamingBlocks},
		{scope: streamScopeTransient, blocks: e.streamingTransient},
		{scope: streamScopeRTransient, blocks: e.streamingRTransient},
	}
}

func (e *Engine) resolveStreamingTransientBlocks() []*config.Block {
	if e.LayoutConfig == nil {
		return nil
	}

	if e.shouldInlineTransientRPrompt() {
		return e.composeLayoutBlocks(e.LayoutConfig.TransientPrompt, e.LayoutConfig.TransientRPrompt, true)
	}

	if len(e.LayoutConfig.TransientPrompt) == 0 {
		return nil
	}

	blocks := make([]*config.Block, 0, len(e.LayoutConfig.TransientPrompt))
	for i := range e.LayoutConfig.TransientPrompt {
		blocks = append(blocks, e.layoutBlock(&e.LayoutConfig.TransientPrompt[i], config.Prompt, config.Left, i != 0))
	}

	return blocks
}

func (e *Engine) resolveStreamingRTransientBlocks() []*config.Block {
	if e.LayoutConfig == nil || len(e.LayoutConfig.TransientRPrompt) == 0 {
		return nil
	}

	if e.shouldInlineTransientRPrompt() {
		block := e.layoutBlock(&e.LayoutConfig.TransientRPrompt[len(e.LayoutConfig.TransientRPrompt)-1], config.RPrompt, config.Right, false)
		return []*config.Block{block}
	}

	block := e.layoutBlock(&e.LayoutConfig.TransientRPrompt[0], config.RPrompt, config.Right, false)
	return []*config.Block{block}
}

func (e *Engine) renderBlockWithText(block *config.Block, blockText string, length int, cancelNewline bool) bool {
	if !block.Force && length == 0 {
		return false
	}

	defer func() {
		e.applyPowerShellBleedPatch()
	}()

	switch block.Type {
	case config.Prompt:
		if block.Newline && !cancelNewline {
			e.writeNewline()
		}

		if block.Alignment == config.Left {
			e.currentLineLength += length
			e.write(blockText)
			return true
		}

		if block.Alignment != config.Right {
			return false
		}

		space, ok := e.canWriteRightBlock(length, false)
		if !ok {
			e.Overflow = block.Overflow

			switch e.Overflow {
			case config.Break:
				e.writeNewline()
			case config.Hide:
				if padText, canFill := e.shouldFill(block.Filler, space+length-e.currentLineLength); canFill {
					e.write(padText)
				}

				e.currentLineLength = 0
				return true
			}
		}

		defer func() {
			e.currentLineLength = 0
			e.Overflow = ""
		}()

		if padText, canFill := e.shouldFill(block.Filler, space); canFill {
			e.write(padText)
			e.write(blockText)
			return true
		}

		if space > 0 {
			e.write(strings.Repeat(" ", space))
		}

		e.write(blockText)
	case config.RPrompt:
		return e.appendRightPromptLine(blockText, length, block.Newline)
	}

	return true
}

func (e *Engine) PendingSegments() map[string]bool {
	e.streamingMu.Lock()
	defer e.streamingMu.Unlock()

	if len(e.pendingSegments) == 0 {
		return map[string]bool{}
	}

	snapshot := make(map[string]bool, len(e.pendingSegments))
	maps.Copy(snapshot, e.pendingSegments)

	return snapshot
}

func (e *Engine) PendingSegmentCount() int {
	e.streamingMu.Lock()
	defer e.streamingMu.Unlock()
	return len(e.pendingSegments)
}
