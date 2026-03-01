package cli

import (
	"bytes"
	"context"
	"testing"
	"time"

	daemonpkg "github.com/jandedobbeleer/oh-my-posh/src/daemon"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime"
	"github.com/stretchr/testify/require"
)

type renderManagedDaemonStub struct {
	initial daemonpkg.RenderResponse
	updates []daemonpkg.RenderResponse
}

func (stub *renderManagedDaemonStub) Stop() {}

func (stub *renderManagedDaemonStub) StartRender(_ daemonpkg.RenderRequest) daemonpkg.RenderResponse {
	return stub.initial
}

func (stub *renderManagedDaemonStub) NextUpdate(_ context.Context, _ string, after uint64) (daemonpkg.RenderResponse, bool) {
	for _, update := range stub.updates {
		if update.Sequence > after {
			return update, true
		}
	}

	return daemonpkg.RenderResponse{}, false
}

func (stub *renderManagedDaemonStub) CompleteSession(_ string) {}

func TestRenderWithDaemonPrintsInitialAndUpdates(t *testing.T) {
	stub := &renderManagedDaemonStub{
		initial: daemonpkg.RenderResponse{
			Type: "initial",
			Bundle: daemonpkg.PromptBundle{
				Primary: "p1",
				RPrompt: "r1",
			},
		},
		updates: []daemonpkg.RenderResponse{
			{
				Type:     "update",
				Sequence: 1,
				Bundle: daemonpkg.PromptBundle{
					Primary:   "p2",
					RPrompt:   "r2",
					Secondary: "s2",
				},
			},
		},
	}
	controller := newDaemonController(func() managedDaemon { return stub })
	out := new(bytes.Buffer)

	err := renderWithDaemon(controller, &runtime.Flags{}, "session-a", false, 2, 20*time.Millisecond, out)
	require.NoError(t, err)
	require.Equal(
		t,
		"primary:p1\nright:r1\nprimary:p2\nright:r2\nsecondary:s2\nstatus:update\nstatus:complete\n",
		out.String(),
	)
}

func TestRenderWithDaemonReturnsErrorWhenStopped(t *testing.T) {
	stub := &renderManagedDaemonStub{
		initial: daemonpkg.RenderResponse{Type: "stopped"},
	}
	controller := newDaemonController(func() managedDaemon { return stub })
	out := new(bytes.Buffer)

	err := renderWithDaemon(controller, &runtime.Flags{}, "session-a", false, 1, 10*time.Millisecond, out)
	require.Error(t, err)
	require.ErrorContains(t, err, "daemon is stopped")
}

func TestWritePromptBundleAlwaysPrintsPrimaryAndRight(t *testing.T) {
	out := new(bytes.Buffer)
	writePromptBundle(out, daemonpkg.PromptBundle{
		Primary: "p",
		RPrompt: "",
	})

	require.Equal(t, "primary:p\nright:\n", out.String())
}

func TestResolveRenderSessionID(t *testing.T) {
	t.Run("explicit session id wins", func(t *testing.T) {
		t.Setenv("POSH_SESSION_ID", "env-session")
		got := resolveRenderSessionID("explicit-session", 1234)
		require.Equal(t, "explicit-session", got)
	})

	t.Run("pid is used when explicit session id is empty", func(t *testing.T) {
		t.Setenv("POSH_SESSION_ID", "env-session")
		got := resolveRenderSessionID("", 4321)
		require.Equal(t, "4321", got)
	})

	t.Run("env session id is used when pid is not set", func(t *testing.T) {
		t.Setenv("POSH_SESSION_ID", "env-session")
		got := resolveRenderSessionID("", 0)
		require.Equal(t, "env-session", got)
	})

	t.Run("default is used when no source is available", func(t *testing.T) {
		t.Setenv("POSH_SESSION_ID", "")
		got := resolveRenderSessionID("", 0)
		require.Equal(t, "default", got)
	})
}
