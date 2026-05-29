package log

import (
	"sync"
	"testing"
	"time"
)

// TestConcurrentLoggingIsRaceFree fires many concurrent log writes and
// concurrent reads of the in-memory buffer. The daemon renders segments
// concurrently and each emits a trace log on completion, so the shared log
// buffer must be safe for concurrent use. Run under `-race` to be
// meaningful; before the buffer was guarded, printLn wrote the global
// strings.Builder outside any lock and this tripped the detector.
func TestConcurrentLoggingIsRaceFree(t *testing.T) {
	prevEnabled, prevRaw := enabled, raw
	t.Cleanup(func() {
		// Cleanup runs single-threaded after Wait, so no lock needed here.
		enabled, raw = prevEnabled, prevRaw
		log.Reset()
	})

	Enable(true) // raw mode: deterministic output, no ANSI escapes

	const writers = 64
	var wg sync.WaitGroup
	wg.Add(writers)
	for range writers {
		go func() {
			defer wg.Done()
			Debug("segment", "value")
			Trace(time.Now(), "segment")
			Error(errSentinel)
			_ = String()
		}()
	}
	wg.Wait()

	if String() == "" {
		t.Fatal("expected log buffer to contain entries after concurrent writes")
	}
}

type sentinelError struct{}

func (sentinelError) Error() string { return "sentinel" }

var errSentinel = sentinelError{}
