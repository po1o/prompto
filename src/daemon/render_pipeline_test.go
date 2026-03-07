package daemon

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/po1o/prompto/src/prompt"
	"github.com/po1o/prompto/src/runtime"

	"github.com/stretchr/testify/require"
)

type rendererStub struct {
	lastPrimary   string
	renderedCalls []string
	callCount     int
	mu            sync.Mutex
}

func (renderer *rendererStub) Bundle(_ *prompt.Engine, primary string, _ bool) PromptBundle {
	renderer.mu.Lock()
	defer renderer.mu.Unlock()

	renderer.callCount++
	renderer.lastPrimary = primary
	renderer.renderedCalls = append(renderer.renderedCalls, primary)
	return PromptBundle{
		Primary: "render",
	}
}

func (renderer *rendererStub) Calls() []string {
	renderer.mu.Lock()
	defer renderer.mu.Unlock()
	out := make([]string, len(renderer.renderedCalls))
	copy(out, renderer.renderedCalls)
	return out
}

func TestRenderPipelineStartRendersInitialBundle(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	sessionRuntime := NewSessionRenderRuntime(registry, nil)
	renderer := &rendererStub{}
	pipeline := NewRenderPipeline(sessionRuntime, renderer, nil)

	bundle, active := pipeline.Start("session-a", &runtime.Flags{}, false)
	require.Equal(t, "render", bundle.Primary)
	require.NotNil(t, active)

	calls := renderer.Calls()
	require.Equal(t, []string{""}, calls)

	active.Complete()
}

func TestRenderPipelineNextRendersAfterUpdate(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	sessionRuntime := NewSessionRenderRuntime(registry, nil)
	renderer := &rendererStub{}
	pipeline := NewRenderPipeline(sessionRuntime, renderer, nil)

	_, active := pipeline.Start("session-a", &runtime.Flags{}, false)
	defer active.Complete()

	go func() {
		time.Sleep(20 * time.Millisecond)
		sessionRuntime.SessionHub("session-a").Publish("path.main")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	update, ok := active.Next(ctx, 0)
	require.True(t, ok)
	require.Equal(t, uint64(1), update.Snapshot.Sequence)
	require.Equal(t, "path.main", update.Snapshot.Payload)
	require.Equal(t, "render", update.Bundle.Primary)

	calls := renderer.Calls()
	require.Equal(t, []string{"", ""}, calls)
}

func TestActiveRenderNextHandlesNil(t *testing.T) {
	var active *ActiveRender
	_, ok := active.Next(context.Background(), 0)
	require.False(t, ok)
}

func TestApplyRenderFlagsNonRepaintUpdatesWorkingDirectory(t *testing.T) {
	term := &runtime.Terminal{}
	term.Init(&runtime.Flags{PWD: "/tmp/first", VimMode: "insert"})
	engine := &prompt.Engine{Env: term}

	applyRenderFlags(engine, &runtime.Flags{PWD: "/tmp/second", VimMode: "normal"}, false)

	require.Equal(t, "/tmp/second", term.Pwd())
	require.Equal(t, "/tmp/second", term.Flags().PWD)
	require.Equal(t, "normal", term.Flags().VimMode)
}

func TestApplyRenderFlagsRepaintOnlyUpdatesVimMode(t *testing.T) {
	term := &runtime.Terminal{}
	term.Init(&runtime.Flags{PWD: "/tmp/first", VimMode: "insert"})
	engine := &prompt.Engine{Env: term}

	applyRenderFlags(engine, &runtime.Flags{PWD: "/tmp/second", VimMode: "normal"}, true)

	require.Equal(t, "/tmp/first", term.Pwd())
	require.Equal(t, "/tmp/first", term.Flags().PWD)
	require.Equal(t, "normal", term.Flags().VimMode)
}
