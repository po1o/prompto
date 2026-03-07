package prompt

import (
	"context"
	"sync"

	"github.com/po1o/prompto/src/config"
)

type sharedExecutionResult struct {
	Source *config.Segment
}

type sharedSegmentProvider interface {
	Execute(ctx context.Context, e *Engine, source *config.Segment) (sharedExecutionResult, bool, error)
}

type sharedProviderFactory func() sharedSegmentProvider

type onceProvider[T any] struct {
	out  T
	err  error
	load func() (T, error)
	once sync.Once
}

func newOnceProvider[T any](load func() (T, error)) *onceProvider[T] {
	return &onceProvider[T]{
		load: load,
	}
}

func (provider *onceProvider[T]) Get() (T, error) {
	provider.once.Do(func() {
		provider.out, provider.err = provider.load()
	})

	return provider.out, provider.err
}

type stateSharedProvider struct{}

func (provider *stateSharedProvider) Execute(ctx context.Context, e *Engine, source *config.Segment) (sharedExecutionResult, bool, error) {
	completed := e.executeSegmentWithContext(ctx, source)
	if !completed {
		return sharedExecutionResult{}, false, nil
	}

	return sharedExecutionResult{
		Source: source,
	}, true, nil
}

func defaultSharedProviderFactories() map[config.SegmentType]sharedProviderFactory {
	factories := make(map[config.SegmentType]sharedProviderFactory, len(config.Segments))
	for segmentType := range config.Segments {
		factories[segmentType] = func() sharedSegmentProvider {
			return &stateSharedProvider{}
		}
	}

	return factories
}

func (e *Engine) resetSharedProviders() {
	e.sharedProviderMu.Lock()
	defer e.sharedProviderMu.Unlock()
	e.sharedProviders = nil
}

func (e *Engine) getOrCreateSharedProvider(
	ctx context.Context,
	segmentType config.SegmentType,
	source *config.Segment,
	factory sharedProviderFactory,
) *onceProvider[sharedExecutionResult] {
	e.sharedProviderMu.Lock()
	defer e.sharedProviderMu.Unlock()

	if e.sharedProviders == nil {
		e.sharedProviders = make(map[config.SegmentType]*onceProvider[sharedExecutionResult])
	}

	if provider, ok := e.sharedProviders[segmentType]; ok {
		return provider
	}

	provider := factory()
	shared := newOnceProvider(func() (sharedExecutionResult, error) {
		res, completed, err := provider.Execute(ctx, e, source)
		if !completed {
			return sharedExecutionResult{}, context.Canceled
		}

		return res, err
	})
	e.sharedProviders[segmentType] = shared
	return shared
}
