package prompt

import (
	"sync"

	"github.com/jandedobbeleer/oh-my-posh/src/config"
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
	load func() (T, error)

	once sync.Once
	out  T
	err  error
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
