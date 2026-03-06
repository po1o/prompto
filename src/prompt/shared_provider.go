package prompt

import (
	"sync"

	"github.com/po1o/prompto/src/config"
)

type sharedExecutionResult struct {
	Text    string
	Enabled bool
}

type sharedSegmentProvider interface {
	Execute(e *Engine, source *config.Segment) (sharedExecutionResult, error)
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

type textSharedProvider struct{}

func (provider *textSharedProvider) Execute(e *Engine, source *config.Segment) (sharedExecutionResult, error) {
	source.Execute(e.Env)
	return sharedExecutionResult{
		Text:    source.Text(),
		Enabled: source.Enabled,
	}, nil
}

func defaultSharedProviderFactories() map[config.SegmentType]sharedProviderFactory {
	return map[config.SegmentType]sharedProviderFactory{
		config.TEXT: func() sharedSegmentProvider {
			return &textSharedProvider{}
		},
	}
}

func (e *Engine) resetSharedProviders() {
	e.sharedProviderMu.Lock()
	defer e.sharedProviderMu.Unlock()
	e.sharedProviders = nil
}

func (e *Engine) getOrCreateSharedProvider(
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
		return provider.Execute(e, source)
	})
	e.sharedProviders[segmentType] = shared
	return shared
}
