package daemon

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/po1o/prompto/src/prompt"
	"github.com/po1o/prompto/src/runtime"
	"github.com/po1o/prompto/src/shell"

	"github.com/stretchr/testify/require"
)

type rendererStub struct {
	lastPrimary   string
	renderedCalls []string
	includeExtras []bool
	callCount     int
	mu            sync.Mutex
}

func (renderer *rendererStub) Bundle(_ *prompt.Engine, primary string, includeExtras bool) PromptBundle {
	renderer.mu.Lock()
	defer renderer.mu.Unlock()

	renderer.callCount++
	renderer.lastPrimary = primary
	renderer.renderedCalls = append(renderer.renderedCalls, primary)
	renderer.includeExtras = append(renderer.includeExtras, includeExtras)

	bundle := PromptBundle{
		Primary: "render",
	}
	if includeExtras {
		bundle.Transient = "transient"
		bundle.RTransient = "rtransient"
	}

	return bundle
}

func (renderer *rendererStub) Calls() []string {
	renderer.mu.Lock()
	defer renderer.mu.Unlock()
	out := make([]string, len(renderer.renderedCalls))
	copy(out, renderer.renderedCalls)
	return out
}

func (renderer *rendererStub) Extras() []bool {
	renderer.mu.Lock()
	defer renderer.mu.Unlock()
	out := make([]bool, len(renderer.includeExtras))
	copy(out, renderer.includeExtras)
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

func TestRenderPipelineRepaintWithoutActiveRenderReturnsNoActiveHandle(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	sessionRuntime := NewSessionRenderRuntime(registry, nil)
	renderer := &rendererStub{}
	pipeline := NewRenderPipeline(sessionRuntime, renderer, nil)

	bundle, active := pipeline.Start("session-a", &runtime.Flags{VimMode: "normal"}, true)

	require.Equal(t, "render", bundle.Primary)
	require.Nil(t, active)

	_, _, ok := registry.GetActiveRender("session-a")
	require.False(t, ok)
}

func TestRenderPipelineReturnsExtrasImmediatelyWhenPrimaryCompletesSynchronously(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "fast-primary.omp.yaml")
	configYAML := `
prompt:
  - segments: ["text.main"]

transient:
  - segments: ["text.transient"]

rtransient:
  - segments: ["text.rtransient"]

text.main:
  type: text
  template: MAIN

text.transient:
  type: text
  template: TL

text.rtransient:
  type: text
  template: TR
`
	require.NoError(t, os.WriteFile(configPath, []byte(configYAML), 0o644))

	registry := NewEngineRegistry(prompt.New)
	sessionRuntime := NewSessionRenderRuntime(registry, nil)
	renderer := &rendererStub{}
	pipeline := NewRenderPipeline(sessionRuntime, renderer, nil)

	flags := &runtime.Flags{
		ConfigPath:    configPath,
		Shell:         shell.GENERIC,
		TerminalWidth: 80,
		Plain:         true,
	}

	bundle, active := pipeline.Start("session-a", flags, false)

	require.Equal(t, "render", bundle.Primary)
	require.Equal(t, "transient", bundle.Transient)
	require.Equal(t, "rtransient", bundle.RTransient)
	require.Nil(t, active)
	require.Equal(t, []bool{true}, renderer.Extras())

	_, _, ok := registry.GetActiveRender("session-a")
	require.False(t, ok)
}
