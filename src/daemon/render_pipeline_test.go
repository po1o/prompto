package daemon

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/prompt"
	"github.com/po1o/prompto/src/runtime"
	"github.com/po1o/prompto/src/shell"

	"github.com/stretchr/testify/require"
)

type rendererStub struct {
	lastPrimary   string
	renderedCalls []string
	options       []bundleOptions
	callCount     int
	mu            sync.Mutex
}

func (renderer *rendererStub) Bundle(_ *prompt.Engine, primary string, options bundleOptions) PromptBundle {
	renderer.mu.Lock()
	defer renderer.mu.Unlock()

	renderer.callCount++
	renderer.lastPrimary = primary
	renderer.renderedCalls = append(renderer.renderedCalls, primary)
	renderer.options = append(renderer.options, options)

	bundle := PromptBundle{
		Primary: "render",
	}
	if options.includeTransient {
		bundle.Transient = "transient"
		bundle.RTransient = "rtransient"
	}
	if options.includeSecondary {
		bundle.Secondary = "secondary"
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

func (renderer *rendererStub) Options() []bundleOptions {
	renderer.mu.Lock()
	defer renderer.mu.Unlock()
	out := make([]bundleOptions, len(renderer.options))
	copy(out, renderer.options)
	return out
}

func TestRenderPipelineStartRendersInitialBundle(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	renderer := &rendererStub{}
	pipeline := NewRenderPipeline(registry, nil, renderer, nil)

	bundle, active := pipeline.Start(RenderRequest{SessionID: "session-a", Flags: &runtime.Flags{}, Cancel: CancelHard})
	require.Equal(t, "render", bundle.Primary)
	require.Equal(t, "transient", bundle.Transient)
	require.Equal(t, "rtransient", bundle.RTransient)
	require.NotNil(t, active)

	calls := renderer.Calls()
	require.Equal(t, []string{""}, calls)
	require.Equal(t, []bundleOptions{{includeTransient: true}}, renderer.Options())

	active.Complete()
}

func TestRenderPipelineNextRendersAfterUpdate(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	renderer := &rendererStub{}
	pipeline := NewRenderPipeline(registry, nil, renderer, nil)

	_, active := pipeline.Start(RenderRequest{SessionID: "session-a", Flags: &runtime.Flags{}, Cancel: CancelHard})
	defer active.Complete()

	go func() {
		time.Sleep(20 * time.Millisecond)
		pipeline.SessionHub("session-a").Publish("path.main")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	update, ok := active.Next(ctx, 0)
	require.True(t, ok)
	require.Equal(t, uint64(1), update.Snapshot.Sequence)
	require.Equal(t, "path.main", update.Snapshot.Payload)
	require.Equal(t, "render", update.Bundle.Primary)
	require.Equal(t, "transient", update.Bundle.Transient)
	require.Equal(t, "rtransient", update.Bundle.RTransient)

	calls := renderer.Calls()
	require.Equal(t, []string{"", ""}, calls)
	require.Equal(t, []bundleOptions{
		{includeTransient: true},
		{includeTransient: true},
	}, renderer.Options())
}

func TestActiveRenderNextHandlesNil(t *testing.T) {
	var active *ActiveRender
	_, ok := active.Next(context.Background(), 0)
	require.False(t, ok)
}

func TestApplyRenderFlagsNonRepaintUpdatesWorkingDirectory(t *testing.T) {
	firstPwd := filepath.Join(t.TempDir(), "first")
	secondPwd := filepath.Join(t.TempDir(), "second")
	term := &runtime.Terminal{}
	term.Init(&runtime.Flags{PWD: firstPwd, VimMode: "insert"})
	engine := &prompt.Engine{Env: term}

	applyRenderFlags(engine, &runtime.Flags{PWD: secondPwd, VimMode: "normal"}, nil, false)

	require.Equal(t, secondPwd, term.Pwd())
	require.Equal(t, secondPwd, term.Flags().PWD)
	require.Equal(t, "normal", term.Flags().VimMode)
}

func TestApplyRenderFlagsRepaintOnlyUpdatesVimMode(t *testing.T) {
	firstPwd := filepath.Join(t.TempDir(), "first")
	secondPwd := filepath.Join(t.TempDir(), "second")
	term := &runtime.Terminal{}
	term.Init(&runtime.Flags{PWD: firstPwd, VimMode: "insert"})
	engine := &prompt.Engine{Env: term}

	applyRenderFlags(engine, &runtime.Flags{PWD: secondPwd, VimMode: "normal"}, nil, true)

	require.Equal(t, firstPwd, term.Pwd())
	require.Equal(t, firstPwd, term.Flags().PWD)
	require.Equal(t, "normal", term.Flags().VimMode)
}

func TestApplyRenderFlagsResolvesEnvFromClientRequest(t *testing.T) {
	term := &runtime.Terminal{}
	term.Init(&runtime.Flags{})
	engine := &prompt.Engine{Env: term}

	// The daemon was started from an SSH session; the local client shell
	// sends its complete environ, which does NOT contain SSH_CONNECTION.
	t.Setenv("SSH_CONNECTION", "10.0.0.40 55103 10.0.0.42 22")
	clientEnv := map[string]string{"CLIENT_ONLY": "from-client"}

	applyRenderFlags(engine, &runtime.Flags{}, clientEnv, false)

	require.IsType(t, &Environment{}, engine.Env)
	// A non-nil client env map is authoritative: a key absent from it is
	// unset in the client shell, with no per-key fallback to the daemon env.
	require.Empty(t, engine.Env.Getenv("SSH_CONNECTION"))
	require.Equal(t, "from-client", engine.Env.Getenv("CLIENT_ONLY"))

	// A request without an env map (nil) falls back to the daemon's own env.
	applyRenderFlags(engine, &runtime.Flags{}, nil, false)
	require.Equal(t, "10.0.0.40 55103 10.0.0.42 22", engine.Env.Getenv("SSH_CONNECTION"))
	require.Empty(t, engine.Env.Getenv("CLIENT_ONLY"))
}

func TestApplyRenderFlagsRepaintRefreshesEnv(t *testing.T) {
	term := &runtime.Terminal{}
	term.Init(&runtime.Flags{VimMode: "insert"})
	engine := &prompt.Engine{Env: term}

	applyRenderFlags(engine, &runtime.Flags{}, map[string]string{"KEY": "hard"}, false)
	applyRenderFlags(engine, &runtime.Flags{VimMode: "normal"}, map[string]string{"KEY": "soft"}, true)

	require.Equal(t, "soft", engine.Env.Getenv("KEY"))
	require.Equal(t, "normal", engine.Env.Flags().VimMode)
}

func TestRenderPipelineRepaintWithoutActiveRenderReturnsNoActiveHandle(t *testing.T) {
	registry := NewEngineRegistry(func(_ *runtime.Flags) *prompt.Engine {
		return &prompt.Engine{}
	})
	renderer := &rendererStub{}
	pipeline := NewRenderPipeline(registry, nil, renderer, nil)

	bundle, active := pipeline.Start(RenderRequest{SessionID: "session-a", Flags: &runtime.Flags{VimMode: "normal"}, Cancel: CancelSoft})

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
	renderer := &rendererStub{}
	pipeline := NewRenderPipeline(registry, nil, renderer, nil)

	flags := &runtime.Flags{
		ConfigPath:    configPath,
		Shell:         shell.GENERIC,
		TerminalWidth: 80,
		Plain:         true,
	}

	bundle, active := pipeline.Start(RenderRequest{SessionID: "session-a", Flags: flags, Cancel: CancelHard})

	require.Equal(t, "render", bundle.Primary)
	require.Equal(t, "transient", bundle.Transient)
	require.Equal(t, "rtransient", bundle.RTransient)
	require.Nil(t, active)
	require.Equal(t, []bundleOptions{{
		includeSecondary: true,
		includeTransient: true,
	}}, renderer.Options())

	_, _, ok := registry.GetActiveRender("session-a")
	require.False(t, ok)
}

func TestRenderPipelineRefreshesTemplateGlobalsForReusedPrimaryEngine(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "status.omp.yaml")
	configYAML := `
prompt:
  - segments: ["status"]

status:
  type: status
  options:
    always_enabled: true
  template: '{{ if gt .Code 0 }}ERROR{{ else }}OK{{ end }}'
`
	require.NoError(t, os.WriteFile(configPath, []byte(configYAML), 0o644))

	registry := NewEngineRegistry(prompt.New)
	pipeline := NewRenderPipeline(registry, nil, nil, nil)

	flags := func(code int) *runtime.Flags {
		return &runtime.Flags{
			ConfigPath:    configPath,
			Shell:         shell.GENERIC,
			TerminalWidth: 80,
			Plain:         true,
			ErrorCode:     code,
		}
	}

	success, active := pipeline.Start(RenderRequest{SessionID: "session-a", Flags: flags(0), Cancel: CancelHard})
	require.Nil(t, active)
	require.True(t, strings.Contains(success.Primary, "OK"))
	require.False(t, strings.Contains(success.Primary, "ERROR"))

	failure, active := pipeline.Start(RenderRequest{SessionID: "session-a", Flags: flags(1), Cancel: CancelHard})
	require.Nil(t, active)
	require.True(t, strings.Contains(failure.Primary, "ERROR"))
}

// TestRenderMarkerDeadlineStaysAheadOfTheClient covers the ordering the timed-out
// marker depends on. The configured deadline and the client's stream deadline
// are set independently, so nothing stops them being equal, or the configured
// one being later. Either way the client hangs up first, the marker is produced
// for nobody, and the shell keeps the pending placeholders — the exact failure
// the marker exists to prevent. The client's deadline is therefore the ceiling.
func TestRenderMarkerDeadlineStaysAheadOfTheClient(t *testing.T) {
	t.Run("no client deadline leaves the configured one alone", func(t *testing.T) {
		require.Equal(t, 60*time.Second, renderMarkerDeadline(60*time.Second, time.Time{}))
	})

	t.Run("a configured deadline that fits is kept", func(t *testing.T) {
		client := time.Now().Add(2 * time.Minute)
		require.Equal(t, 30*time.Second, renderMarkerDeadline(30*time.Second, client))
	})

	t.Run("a configured deadline past the client is pulled in", func(t *testing.T) {
		client := time.Now().Add(20 * time.Second)
		marker := renderMarkerDeadline(5*time.Minute, client)

		// Bounded on both sides: pulling it in must leave the margin, and must
		// not collapse it to nothing.
		require.Greater(t, marker, 14*time.Second)
		require.LessOrEqual(t, marker, 20*time.Second-clientReportMargin)
	})

	t.Run("deadlines set to the same value still leave room to report", func(t *testing.T) {
		// The case that has no margin by construction: both sides asked for the
		// same 30 seconds.
		client := time.Now().Add(30 * time.Second)
		marker := renderMarkerDeadline(30*time.Second, client)

		require.Less(t, marker, 30*time.Second, "the marker must land before the client gives up")
	})

	t.Run("no deadline configured still reports before the client goes", func(t *testing.T) {
		client := time.Now().Add(20 * time.Second)
		marker := renderMarkerDeadline(0, client)

		require.Greater(t, marker, 14*time.Second)
		require.LessOrEqual(t, marker, 20*time.Second-clientReportMargin)
	})

	t.Run("a negative deadline defers to the client rather than disabling", func(t *testing.T) {
		// Through the real mapping, not a hand-written zero: this is the
		// behaviour a user writing `render_timeout: -1` actually gets.
		configured := (&config.Config{RenderTimeout: -1}).GetRenderTimeout()
		require.Equal(t, time.Duration(0), configured, "a negative config means no deadline of its own")

		client := time.Now().Add(20 * time.Second)
		marker := renderMarkerDeadline(configured, client)

		// It cannot mean "never": the client stops listening regardless, and
		// silence there is the bug the marker exists to prevent.
		require.Greater(t, marker, time.Duration(0),
			"deferring must still leave a marker the client can receive")
		require.LessOrEqual(t, marker, 20*time.Second-clientReportMargin)
	})

	t.Run("a client already out of time gets nothing", func(t *testing.T) {
		require.Equal(t, time.Duration(0), renderMarkerDeadline(time.Minute, time.Now()))
	})
}
