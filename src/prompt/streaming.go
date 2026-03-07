package prompt

import (
	"context"
	"fmt"
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

func segmentKey(blockIndex, segmentIndex int, segment *config.Segment) string {
	return fmt.Sprintf("%d:%d:%s", blockIndex, segmentIndex, segment.Name())
}

type streamingSegment struct {
	segment *config.Segment
	key     string
}

type streamingResult struct {
	segment *config.Segment
	key     string
}

// PrimaryStreaming renders a prompt with a timeout cutoff and pending placeholders.
func (e *Engine) PrimaryStreaming(ctx context.Context, timeout time.Duration, updateCallback func(string)) string {
	if timeout <= 0 {
		timeout = 100 * time.Millisecond
	}

	if ctx == nil {
		ctx = context.Background()
	}

	e.resetSharedProviders()

	e.pendingSegments = make(map[string]bool)
	e.cachedValues = make(map[string]string)
	e.segmentCacheKeys = make(map[string]string)
	e.streamingBlocks = e.resolveStreamingBlocks()

	segmentsToExecute, completed := e.prepareStreamingSegments()
	results := e.startStreamingExecutions(ctx, segmentsToExecute)
	e.collectStreamingResultsUntil(ctx, timeout, results, completed)

	e.streamingMu.Lock()
	for _, entry := range segmentsToExecute {
		if completed[entry.key] {
			continue
		}

		e.pendingSegments[entry.key] = true
	}
	e.streamingMu.Unlock()

	initialPrompt := e.renderStreamingPrompt()

	if len(e.pendingSegments) > 0 {
		go func() {
			for result := range results {
				if ctx.Err() != nil {
					return
				}

				e.streamingMu.Lock()
				delete(e.pendingSegments, result.key)
				e.streamingMu.Unlock()
				updateCallback(result.segment.Name())
			}
			if ctx.Err() == nil {
				updateCallback("")
			}
		}()
	}

	return initialPrompt
}

func (e *Engine) prepareStreamingSegments() ([]streamingSegment, map[string]bool) {
	segmentsToExecute := make([]streamingSegment, 0, 32)
	completed := make(map[string]bool)

	blocks := e.streamingBlocks

	for blockIndex, block := range blocks {
		for segmentIndex, segment := range block.Segments {
			key := segmentKey(blockIndex, segmentIndex, segment)
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
				key:     key,
			})
		}
	}

	return segmentsToExecute, completed
}

func (e *Engine) startStreamingExecutions(ctx context.Context, segments []streamingSegment) <-chan streamingResult {
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

		sources[entry.segment.Type] = entry.segment.Clone()
	}

	for _, entry := range segments {
		wg.Add(1)
		go func(entry streamingSegment) {
			defer wg.Done()
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
	select {
	case <-ctx.Done():
		return
	default:
	}

	segment := entry.segment
	e.markSegmentPending(segment)

	if providerFactory, ok := e.sharedProviderFactory[segment.Type]; ok {
		if err := segment.MapSegmentWithWriter(e.Env); err == nil {
			sharedProvider := e.getOrCreateSharedProvider(segment.Type, sources[segment.Type], providerFactory)
			if res, sharedErr := sharedProvider.Get(); sharedErr == nil {
				_ = segment.CopyWriterStateFrom(res.Source)
			}
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		e.markSegmentDone(segment)
		results <- streamingResult{segment: segment, key: entry.key}
		return
	}

	completed := e.executeSegmentWithContext(ctx, segment)
	if !completed {
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	e.markSegmentDone(segment)
	results <- streamingResult{segment: segment, key: entry.key}
}

func (e *Engine) executeSegmentWithContext(ctx context.Context, segment *config.Segment) bool {
	done := make(chan struct{})
	gidChan := make(chan uint64, 1)

	go func() {
		gidChan <- runjobs.CurrentGID()
		e.executeWithoutLegacySegmentCache(segment)
		close(done)
	}()

	gid := <-gidChan

	if segment.Timeout <= 0 {
		select {
		case <-done:
			return true
		case <-ctx.Done():
			_ = runjobs.KillGoroutineChildren(gid)
			return false
		}
	}

	select {
	case <-done:
		return true
	case <-ctx.Done():
		_ = runjobs.KillGoroutineChildren(gid)
		return false
	case <-time.After(time.Duration(segment.Timeout) * time.Millisecond):
		log.Errorf("timeout after %dms for segment: %s", segment.Timeout, segment.Name())
		if err := runjobs.KillGoroutineChildren(gid); err != nil {
			log.Errorf("failed to kill child processes for goroutine %d (segment: %s): %v", gid, segment.Name(), err)
		}
		return true
	}
}

func (e *Engine) collectStreamingResultsUntil(
	ctx context.Context,
	timeout time.Duration,
	results <-chan streamingResult,
	completed map[string]bool,
) {
	timeoutChan := time.After(timeout)
	doneWaiting := false
	for !doneWaiting {
		select {
		case result, ok := <-results:
			if !ok {
				doneWaiting = true
				continue
			}
			completed[result.key] = true
		case <-ctx.Done():
			doneWaiting = true
		case <-timeoutChan:
			doneWaiting = true
		}
	}
}

// PrimaryRepaint re-renders prompt state for repaint without starting new computations.
// Only the vim segment is re-executed. Other segments are served from completed or pending cache state.
func (e *Engine) PrimaryRepaint() string {
	if e.pendingSegments == nil {
		e.pendingSegments = make(map[string]bool)
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

	blocks := e.streamingBlocks

	for blockIndex, block := range blocks {
		for segmentIndex, segment := range block.Segments {
			key := segmentKey(blockIndex, segmentIndex, segment)
			if segment.Type == config.SegmentType("vim") {
				_ = segment.MapSegmentWithWriter(e.Env)
				segment.Execute(e.Env)
				continue
			}

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

	if e.Config.FinalSpace {
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
	blockText, length := e.writeBlockSegmentsStreaming(block, blockIndex)
	return e.renderBlockWithText(block, blockText, length, cancelNewline)
}

func (e *Engine) writeBlockSegmentsStreaming(block *config.Block, blockIndex int) (string, int) {
	segmentIndex := 0

	type pendingRestore struct {
		segment             *config.Segment
		text                string
		background          color.Ansi
		backgroundTemplates template.List
		enabled             bool
	}

	restores := make([]pendingRestore, 0, len(block.Segments))

	for segmentPosition, segment := range block.Segments {
		key := segmentKey(blockIndex, segmentPosition, segment)
		if e.pendingSegments[key] {
			cachedVal := e.cachedValues[key]
			enabled, text, background := segment.GetPendingText(cachedVal, e.Config)
			if !enabled {
				continue
			}

			restores = append(restores, pendingRestore{
				segment:             segment,
				text:                segment.Text(),
				background:          segment.Background,
				backgroundTemplates: segment.BackgroundTemplates,
				enabled:             segment.Enabled,
			})

			segment.SetText(text)
			if background != "" {
				segment.Background = background
			} else {
				segment.Background = "darkGray"
			}
			segment.BackgroundTemplates = nil
			segment.Enabled = true

			e.setActiveSegment(segment)
			e.renderActiveSegment()
			continue
		}

		if segment.Text() != "" && segment.Enabled {
			origForegroundTemplates := segment.ForegroundTemplates
			origBackgroundTemplates := segment.BackgroundTemplates
			segment.ForegroundTemplates = nil
			segment.BackgroundTemplates = nil
			segmentIndex++
			e.writeSegment(block, segment)
			segment.ForegroundTemplates = origForegroundTemplates
			segment.BackgroundTemplates = origBackgroundTemplates
			continue
		}

		if !segment.Render(segmentIndex, e.forceRender) {
			continue
		}

		segmentIndex++
		renderedAt := time.Now()
		e.markSegmentRendered(segment, renderedAt)
		e.storeSegmentCache(segment, renderedAt)
		e.writeSegment(block, segment)
	}

	if e.activeSegment != nil && len(block.TrailingDiamond) > 0 {
		e.activeSegment.TrailingDiamond = block.TrailingDiamond
	}

	e.writeSeparator(true)

	for _, restore := range restores {
		restore.segment.SetText(restore.text)
		restore.segment.Background = restore.background
		restore.segment.BackgroundTemplates = restore.backgroundTemplates
		restore.segment.Enabled = restore.enabled
	}

	e.activeSegment = nil
	e.previousActiveSegment = nil

	return terminal.String()
}

func (e *Engine) renderBlockWithText(block *config.Block, blockText string, length int, cancelNewline bool) bool {
	if !block.Force && length == 0 {
		return false
	}

	defer func() {
		e.applyPowerShellBleedPatch()
	}()

	if block.Newline && !cancelNewline {
		e.writeNewline()
	}

	switch block.Type {
	case config.Prompt:
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
		e.rprompt = blockText
		e.rpromptLength = length
	}

	return true
}

func (e *Engine) StreamingMu() *sync.Mutex {
	return &e.streamingMu
}

func (e *Engine) PendingSegments() map[string]bool {
	return e.pendingSegments
}
