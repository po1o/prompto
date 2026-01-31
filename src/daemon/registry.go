package daemon

import (
	"context"
	"sync"
	"time"

	"github.com/jandedobbeleer/oh-my-posh/src/prompt"
)

// Future represents an in-flight computation that can be shared across requests.
type Future struct {
	done     chan struct{}
	result   any
	err      error
	cancel   context.CancelFunc
	refCount int
	mu       sync.Mutex
}

// Wait blocks until the computation completes or ctx is cancelled.
func (f *Future) Wait(ctx context.Context) (any, error) {
	select {
	case <-f.done:
		return f.result, f.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Done returns a channel that closes when the computation completes.
func (f *Future) Done() <-chan struct{} {
	return f.done
}

// Result returns the computation result. Only valid after Done() closes.
func (f *Future) Result() (any, error) {
	return f.result, f.err
}

// ComputationRegistry manages in-flight segment computations.
// It decouples computation lifecycle from request lifecycle so:
// - Soft Cancel: new requests reuse existing computations (vim mode toggle).
// - Hard Cancel: computations are aborted to prevent stale cache writes (new command).
type ComputationRegistry struct {
	// computations maps sessionID -> (segmentName -> Future)
	computations map[string]map[string]*Future
	mu           sync.Mutex
}

// NewComputationRegistry creates a new registry.
func NewComputationRegistry() *ComputationRegistry {
	return &ComputationRegistry{
		computations: make(map[string]map[string]*Future),
	}
}

// GetOrCreate returns an existing Future for the segment, or creates one using computeFn.
// The computeFn receives a context that will be cancelled on Hard Cancel.
// Returns the Future and whether it was newly created (true) or reused (false).
func (r *ComputationRegistry) GetOrCreate(sessionID, segmentName string, computeFn func(ctx context.Context) (any, error)) (*Future, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Get or create session map
	sessionMap, ok := r.computations[sessionID]
	if !ok {
		sessionMap = make(map[string]*Future)
		r.computations[sessionID] = sessionMap
	}

	// Check for existing computation
	if future, ok := sessionMap[segmentName]; ok {
		future.mu.Lock()
		future.refCount++
		future.mu.Unlock()
		return future, false
	}

	// Create new computation
	ctx, cancel := context.WithCancel(context.Background())
	future := &Future{
		done:     make(chan struct{}),
		cancel:   cancel,
		refCount: 1,
	}
	sessionMap[segmentName] = future

	// Run computation in background
	go func() {
		result, err := computeFn(ctx)
		future.mu.Lock()
		future.result = result
		future.err = err
		future.mu.Unlock()
		close(future.done)

		// Clean up after completion (with delay to allow late subscribers)
		time.AfterFunc(5*time.Second, func() {
			r.cleanup(sessionID, segmentName)
		})
	}()

	return future, true
}

// Get returns an existing Future if present, nil otherwise.
func (r *ComputationRegistry) Get(sessionID, segmentName string) *Future {
	r.mu.Lock()
	defer r.mu.Unlock()

	if sessionMap, ok := r.computations[sessionID]; ok {
		if future, ok := sessionMap[segmentName]; ok {
			future.mu.Lock()
			future.refCount++
			future.mu.Unlock()
			return future
		}
	}
	return nil
}

// Release decrements the refCount for a Future.
func (r *ComputationRegistry) Release(sessionID, segmentName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if sessionMap, ok := r.computations[sessionID]; ok {
		if future, ok := sessionMap[segmentName]; ok {
			future.mu.Lock()
			future.refCount--
			future.mu.Unlock()
		}
	}
}

// HardCancel aborts all computations for a session.
// Used when context changes (new command) - stale computations must not write to cache.
func (r *ComputationRegistry) HardCancel(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if sessionMap, ok := r.computations[sessionID]; ok {
		for _, future := range sessionMap {
			future.cancel()
		}
		delete(r.computations, sessionID)
	}
}

// SoftCancel does not stop computations.
// The RPC stream is cancelled separately; computations remain for reuse.
func (r *ComputationRegistry) SoftCancel(_ string) {
	// No-op: computations continue, will be reused by next request
}

// GetPendingFutures returns all futures for a session that haven't completed yet.
// Used by repaint to wait for ongoing computations from the previous request.
func (r *ComputationRegistry) GetPendingFutures(sessionID string) []*Future {
	r.mu.Lock()
	defer r.mu.Unlock()

	var pending []*Future
	if sessionMap, ok := r.computations[sessionID]; ok {
		for _, future := range sessionMap {
			select {
			case <-future.done:
				// Already done, skip
			default:
				// Still pending
				future.mu.Lock()
				future.refCount++
				future.mu.Unlock()
				pending = append(pending, future)
			}
		}
	}
	return pending
}

// CleanSession removes all entries for a session (called on session end).
func (r *ComputationRegistry) CleanSession(sessionID string) {
	r.HardCancel(sessionID)
}

// cleanup removes a completed computation after delay.
func (r *ComputationRegistry) cleanup(sessionID, segmentName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if sessionMap, ok := r.computations[sessionID]; ok {
		if future, ok := sessionMap[segmentName]; ok {
			future.mu.Lock()
			refCount := future.refCount
			future.mu.Unlock()

			// Only delete if no active references
			if refCount <= 0 {
				delete(sessionMap, segmentName)
				if len(sessionMap) == 0 {
					delete(r.computations, sessionID)
				}
			}
		}
	}
}

// GetPendingKeys returns all segment names with pending computations for a session.
func (r *ComputationRegistry) GetPendingKeys(sessionID string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var keys []string
	if sessionMap, ok := r.computations[sessionID]; ok {
		for key, future := range sessionMap {
			select {
			case <-future.done:
				// Already done, skip
			default:
				keys = append(keys, key)
			}
		}
	}
	return keys
}

// SessionComputationRegistry wraps ComputationRegistry for a specific session.
// It implements prompt.ComputationRegistry for use inside the prompt engine.
type SessionComputationRegistry struct {
	registry  *ComputationRegistry
	sessionID string
}

// NewSessionRegistry creates a session-scoped registry wrapper.
func (r *ComputationRegistry) NewSessionRegistry(sessionID string) *SessionComputationRegistry {
	return &SessionComputationRegistry{
		registry:  r,
		sessionID: sessionID,
	}
}

// GetOrStart implements prompt.ComputationRegistry.
func (s *SessionComputationRegistry) GetOrStart(segmentName string, computeFn func() prompt.SegmentResult) (<-chan struct{}, bool) {
	future, isNew := s.registry.GetOrCreate(s.sessionID, segmentName, func(ctx context.Context) (any, error) {
		// Check if cancelled before starting
		if ctx.Err() != nil {
			return prompt.SegmentResult{}, ctx.Err()
		}
		result := computeFn()
		// Check if cancelled after completion (result won't be used)
		if ctx.Err() != nil {
			return prompt.SegmentResult{}, ctx.Err()
		}
		return result, nil
	})
	return future.Done(), isNew
}

// GetResult implements prompt.ComputationRegistry.
func (s *SessionComputationRegistry) GetResult(segmentName string) (prompt.SegmentResult, bool) {
	future := s.registry.Get(s.sessionID, segmentName)
	if future == nil {
		return prompt.SegmentResult{}, false
	}
	defer s.registry.Release(s.sessionID, segmentName)

	select {
	case <-future.Done():
		result, err := future.Result()
		if err != nil {
			return prompt.SegmentResult{}, false
		}
		if sr, ok := result.(prompt.SegmentResult); ok {
			return sr, true
		}
		return prompt.SegmentResult{}, false
	default:
		// Not complete yet
		return prompt.SegmentResult{}, false
	}
}

// IsPending implements prompt.ComputationRegistry.
func (s *SessionComputationRegistry) IsPending(segmentName string) bool {
	future := s.registry.Get(s.sessionID, segmentName)
	if future == nil {
		return false
	}
	defer s.registry.Release(s.sessionID, segmentName)

	select {
	case <-future.Done():
		return false
	default:
		return true
	}
}

// GetPendingKeys implements prompt.ComputationRegistry.
func (s *SessionComputationRegistry) GetPendingKeys() []string {
	return s.registry.GetPendingKeys(s.sessionID)
}
