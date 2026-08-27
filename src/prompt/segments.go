package prompt

import (
	"context"
	"time"

	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/log"
	runjobs "github.com/po1o/prompto/src/runtime/jobs"
	"github.com/po1o/prompto/src/terminal"
)

type result struct {
	segment *config.Segment
	index   int
}

func (e *Engine) writeBlockSegments(block *config.Block) (string, int) {
	length := len(block.Segments)

	if length == 0 {
		return "", 0
	}

	e.prepareSegmentStates(block.Segments, false)

	out := make(chan result, length)

	e.writeSegmentsConcurrently(block.Segments, out)

	e.writeSegments(out, block)

	e.writeBlockSeparator(block)

	e.endBlock()

	return terminal.String()
}

// writeSegmentsConcurrently uses individual goroutines for each segment
func (e *Engine) writeSegmentsConcurrently(segments []*config.Segment, out chan result) {
	sources := make(map[config.SegmentType]*config.Segment)
	for _, segment := range segments {
		if _, ok := e.sharedProviderFactory[segment.Type]; !ok {
			continue
		}

		if _, ok := sources[segment.Type]; ok {
			continue
		}

		sources[segment.Type] = segment.Clone()
	}

	for i, segment := range segments {
		go func(segment *config.Segment, index int) {
			e.markSegmentPending(segment)

			if e.applySegmentCacheBeforeExecute(segment) {
				out <- result{segment, index}
				return
			}

			if providerFactory, ok := e.sharedProviderFactory[segment.Type]; ok {
				err := segment.MapSegmentWithWriter(e.Env)
				if err != nil {
					e.markSegmentDone(segment)
					e.notifySegmentUpdate(segment.Name())
					out <- result{segment, index}
					return
				}

				sharedProvider := e.getOrCreateSharedProvider(context.Background(), segment.Type, sources[segment.Type], providerFactory)

				res, sharedErr := sharedProvider.Get()
				if sharedErr == nil {
					_ = segment.CopyWriterStateFrom(res.Source)
				}

				e.markSegmentDone(segment)
				e.notifySegmentUpdate(segment.Name())
				out <- result{segment, index}
				return
			}

			if segment.Timeout > 0 {
				e.executeSegmentWithTimeout(segment)
			} else {
				e.executeWithoutLegacySegmentCache(segment)
			}

			e.markSegmentDone(segment)
			e.notifySegmentUpdate(segment.Name())
			out <- result{segment, index}
		}(segment, i)
	}
}

// executeSegmentWithTimeout handles segment execution with timeout logic
func (e *Engine) executeSegmentWithTimeout(segment *config.Segment) {
	done := make(chan bool)
	gidChan := make(chan uint64, 1)

	go func() {
		// Get GID after segment.Execute creates the Job
		gidChan <- runjobs.CurrentGID()
		segment.Execute(e.Env)
		done <- true
	}()

	// Wait for the GID to be available
	gid := <-gidChan

	select {
	case <-done:
		// Completed before timeout
	case <-time.After(time.Duration(segment.Timeout) * time.Millisecond):
		log.Errorf("timeout after %dms for segment: %s", segment.Timeout, segment.Name())

		if err := runjobs.KillGoroutineChildren(gid); err != nil {
			log.Errorf("failed to kill child processes for goroutine %d (segment: %s): %v", gid, segment.Name(), err)
		}
	}
}

func (e *Engine) writeSegments(out chan result, block *config.Block) {
	count := len(block.Segments)
	current := 0
	executedCount := 0
	results := make([]*config.Segment, count)
	// Pre-allocate map with known capacity to reduce allocations
	executed := make(map[string]bool, count)
	segmentIndex := 0

	// Process results as they come in, eliminating busy waiting
	for executedCount < count {
		res := <-out // Block until result is available
		executedCount++

		results[res.index] = res.segment
		executed[res.segment.Name()] = true

		// Process segments that can now be rendered
		for current < count && results[current] != nil {
			segment := results[current]
			if !e.canRenderSegment(segment, executed) {
				break
			}

			if segment.Render(segmentIndex, e.forceRender) {
				segmentIndex++
				renderedAt := time.Now()
				e.markSegmentRendered(segment, renderedAt)
				e.storeSegmentCache(segment, renderedAt)
			}

			e.writeSegment(block, segment)
			current++
		}
	}

	// render all remaining segments where the needs can't be resolved
	for current < executedCount {
		segment := results[current]
		if segment.Render(segmentIndex, e.forceRender) {
			segmentIndex++
			renderedAt := time.Now()
			e.markSegmentRendered(segment, renderedAt)
			e.storeSegmentCache(segment, renderedAt)
		}

		e.writeSegment(block, segment)
		current++
	}
}

// writeKeptSegment draws a segment the caller is about to skip, but only when
// keep_when_empty asks for its shape to be held. Every streaming path that
// bails out on a segment goes through here, so a kept segment cannot silently
// reflow the prompt on one path and hold its place on another.
func (e *Engine) writeKeptSegment(block *config.Block, segment *config.Segment) {
	if !segment.KeepWhenEmpty {
		return
	}

	e.writeSegment(block, segment)
}

// writeBlockSeparator closes the block. A line's own trailing glyph stands in
// for the last segment's, because it is closing the whole line rather than that
// one segment. The substitution is undone afterwards so it cannot outlive the
// block: layoutBlock hands out clones today, but nothing in the signature says
// the segment is ours to keep.
func (e *Engine) writeBlockSeparator(block *config.Block) {
	if e.activeSegment == nil || len(block.TrailingGlyph) == 0 {
		e.writeSeparator(true)
		return
	}

	previous := e.activeSegment.TrailingGlyph
	e.activeSegment.TrailingGlyph = block.TrailingGlyph

	e.writeSeparator(true)

	e.activeSegment.TrailingGlyph = previous
}

func (e *Engine) writeSegment(block *config.Block, segment *config.Segment) {
	if !segment.Enabled && !segment.KeepWhenEmpty {
		return
	}

	if colors, newCycle := cycle.Loop(); colors != nil {
		cycle = &newCycle
		segment.Foreground = colors.Foreground
		segment.Background = colors.Background
	}

	// Same restore as writeBlockSeparator: the block opens the line, but the
	// segment must not carry that glyph into the next render it appears in.
	//
	// The restore is deferred to the end of the block rather than to the end of
	// this call, because the *next* segment reads this one's LeadingGlyph
	// through HasEmptyGlyphAtEnd to decide whether to carve its own opening out
	// of this segment's background. Undoing it here would make a segment the
	// line opened for look like bare text, and the next glyph would float.
	if terminal.Len() == 0 && len(block.LeadingGlyph) > 0 {
		previous := segment.LeadingGlyph
		segment.LeadingGlyph = block.LeadingGlyph

		e.restoreLeadingGlyph = func() {
			segment.LeadingGlyph = previous
		}
	}

	e.setActiveSegment(segment)
	e.renderActiveSegment()
}

// endBlock closes out the per-block engine state. It runs the leading glyph
// restore writeSegment deferred, so the substitution never outlives the block.
func (e *Engine) endBlock() {
	if e.restoreLeadingGlyph != nil {
		e.restoreLeadingGlyph()
		e.restoreLeadingGlyph = nil
	}

	e.activeSegment = nil
	e.previousActiveSegment = nil
}

// canRenderSegment now uses map for O(1) lookups instead of O(n) slice search
func (e *Engine) canRenderSegment(segment *config.Segment, executed map[string]bool) bool {
	for _, name := range segment.Needs {
		if !executed[name] {
			return false
		}
	}

	return true
}
