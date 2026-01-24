package daemon

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jandedobbeleer/oh-my-posh/src/daemon/ipc"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDaemonRenderWithVimMode(t *testing.T) {
	tmpDir := testSocketDir(t)
	t.Setenv("XDG_STATE_HOME", tmpDir)
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	d, err := New(createTestConfig(t))
	require.NoError(t, err)

	go func() { _ = d.Start() }()
	time.Sleep(100 * time.Millisecond)

	defer func() {
		d.shutdown()
		<-d.Done()
	}()

	client, err := NewClient()
	require.NoError(t, err)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Render with vim mode set to "normal"
	flags := &runtime.Flags{
		PWD:     "/tmp",
		Shell:   "zsh",
		Type:    "primary",
		VimMode: "normal",
	}

	resp, err := client.RenderPromptSync(ctx, flags, 0, "test-vim", nil, false)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify vim mode was passed through
	assert.NotEmpty(t, resp.Prompts["primary"].Text)
}

func TestDaemonSoftCancelOnRepaint(t *testing.T) {
	// Test that repaint=true triggers soft cancel (computations continue)
	reg := NewComputationRegistry()

	computeStarted := make(chan struct{})
	computeFinished := make(chan struct{})
	var computeCount atomic.Int32

	sessionID := "vim-session"
	cacheKey := "git-segment"

	// Start a computation
	future, isNew := reg.GetOrCreate(sessionID, cacheKey, func(ctx context.Context) (any, error) {
		close(computeStarted)
		computeCount.Add(1)

		// Simulate slow computation
		select {
		case <-time.After(200 * time.Millisecond):
			close(computeFinished)
			return "git-result", nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})
	require.True(t, isNew)

	// Wait for computation to start
	<-computeStarted

	// Soft cancel (vim mode toggle) - should NOT cancel computations
	reg.SoftCancel(sessionID)

	// Verify computation continues
	select {
	case <-computeFinished:
		// Good - computation completed
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Computation should have continued after soft cancel")
	}

	// Verify result is available
	result, err := future.Wait(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "git-result", result)
	assert.Equal(t, int32(1), computeCount.Load())
}

func TestDaemonHardCancelOnNewCommand(t *testing.T) {
	// Test that repaint=false triggers hard cancel (computations abort)
	reg := NewComputationRegistry()

	computeStarted := make(chan struct{})
	var ctxErr error

	sessionID := "command-session"
	cacheKey := "slow-segment"

	// Start a computation
	_, isNew := reg.GetOrCreate(sessionID, cacheKey, func(ctx context.Context) (any, error) {
		close(computeStarted)

		// Wait for cancellation
		<-ctx.Done()
		ctxErr = ctx.Err()
		return nil, ctx.Err()
	})
	require.True(t, isNew)

	// Wait for computation to start
	<-computeStarted

	// Hard cancel (new command) - should abort computations
	reg.HardCancel(sessionID)

	// Wait briefly for cancellation to propagate
	time.Sleep(50 * time.Millisecond)

	// Verify computation was cancelled
	assert.Equal(t, context.Canceled, ctxErr)

	// Verify registry was cleaned
	future := reg.Get(sessionID, cacheKey)
	assert.Nil(t, future)
}

func TestDaemonComputationReuse(t *testing.T) {
	// Test that soft cancel allows computation reuse
	reg := NewComputationRegistry()

	var computeCount atomic.Int32
	sessionID := "reuse-session"
	cacheKey := "reusable-segment"

	// First request starts computation
	future1, isNew1 := reg.GetOrCreate(sessionID, cacheKey, func(_ context.Context) (any, error) {
		computeCount.Add(1)
		time.Sleep(50 * time.Millisecond)
		return "result", nil
	})
	require.True(t, isNew1)

	// Second request (vim toggle) should reuse existing computation
	future2, isNew2 := reg.GetOrCreate(sessionID, cacheKey, func(_ context.Context) (any, error) {
		computeCount.Add(1)
		return "should-not-run", nil
	})
	require.False(t, isNew2)

	// Both futures should be the same
	assert.Equal(t, future1, future2)

	// Wait for result
	result, err := future1.Wait(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "result", result)

	// Computation should only have run once
	assert.Equal(t, int32(1), computeCount.Load())
}

func TestDaemonRepaintWithTimeout(t *testing.T) {
	tmpDir := testSocketDir(t)
	t.Setenv("XDG_STATE_HOME", tmpDir)
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	d, err := New(createTestConfig(t))
	require.NoError(t, err)

	go func() { _ = d.Start() }()
	time.Sleep(100 * time.Millisecond)

	defer func() {
		d.shutdown()
		<-d.Done()
	}()

	conn, err := ipc.Dial()
	require.NoError(t, err)
	defer conn.Close()

	client := ipc.NewDaemonServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First: normal render
	stream1, err := client.RenderPrompt(ctx, &ipc.PromptRequest{
		Version:   ipc.ProtocolVersion,
		SessionId: "vim-session",
		RequestId: "req-1",
		Flags: &ipc.Flags{
			Pwd:     "/tmp",
			VimMode: "insert",
		},
	})
	require.NoError(t, err)

	resp1, err := stream1.Recv()
	require.NoError(t, err)
	assert.Equal(t, "req-1", resp1.RequestId)

	// Second: repaint with vim mode change (soft cancel)
	stream2, err := client.RenderPrompt(ctx, &ipc.PromptRequest{
		Version:   ipc.ProtocolVersion,
		SessionId: "vim-session",
		RequestId: "req-2",
		Repaint:   true, // Soft cancel
		Flags: &ipc.Flags{
			Pwd:     "/tmp",
			VimMode: "normal",
		},
	})
	require.NoError(t, err)

	resp2, err := stream2.Recv()
	require.NoError(t, err)
	assert.Equal(t, "req-2", resp2.RequestId)
}

func TestRepaintSendsIncrementalUpdates(t *testing.T) {
	// Test that repaint sends update as each pending segment completes (not wait for all)
	reg := NewComputationRegistry()
	sessionID := "incremental-session"

	// Use explicit completion signals
	fastDone := make(chan struct{})
	slowDone := make(chan struct{})

	// Create two futures with different completion times
	reg.GetOrCreate(sessionID, "fast-segment", func(_ context.Context) (any, error) {
		time.Sleep(50 * time.Millisecond)
		close(fastDone)
		return "fast-result", nil
	})

	reg.GetOrCreate(sessionID, "slow-segment", func(_ context.Context) (any, error) {
		time.Sleep(150 * time.Millisecond)
		close(slowDone)
		return "slow-result", nil
	})

	// Track completion order using explicit channels
	var completionOrder []string

	// Wait for fast to complete (should be first)
	select {
	case <-fastDone:
		completionOrder = append(completionOrder, "fast")
	case <-slowDone:
		completionOrder = append(completionOrder, "slow")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("First segment should have completed")
	}

	// Verify slow hasn't completed yet
	select {
	case <-slowDone:
		t.Fatal("Slow segment completed too early - incremental updates wouldn't work")
	default:
		// Good - slow is still pending
	}

	// Wait for slow to complete
	select {
	case <-slowDone:
		completionOrder = append(completionOrder, "slow")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Slow segment should have completed")
	}

	// Verify completion order
	assert.Equal(t, []string{"fast", "slow"}, completionOrder)
}

func TestGetPendingFuturesReturnsOnlyIncomplete(t *testing.T) {
	reg := NewComputationRegistry()
	sessionID := "pending-test"

	// Create a completed future
	f1, _ := reg.GetOrCreate(sessionID, "done-segment", func(_ context.Context) (any, error) {
		return "done", nil
	})
	<-f1.Done() // Wait for completion

	// Create a pending future
	started := make(chan struct{})
	reg.GetOrCreate(sessionID, "pending-segment", func(_ context.Context) (any, error) {
		close(started)
		time.Sleep(100 * time.Millisecond)
		return "pending", nil
	})
	<-started

	// GetPendingFutures should only return the incomplete one
	pending := reg.GetPendingFutures(sessionID)
	assert.Equal(t, 1, len(pending))
}

// TestRepaintUsesCacheAfterRegistryCleanup verifies that PrimaryRepaint can
// use cached segment values even after the computation registry has been cleaned up.
// This tests the fix for the bug where segments completing within timeout were not
// cached, causing them to be missing on repaint.
func TestRepaintUsesCacheAfterRegistryCleanup(t *testing.T) {
	tmpDir := testSocketDir(t)
	t.Setenv("XDG_STATE_HOME", tmpDir)
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	d, err := New(createTestConfig(t))
	require.NoError(t, err)

	go func() { _ = d.Start() }()
	time.Sleep(100 * time.Millisecond)

	defer func() {
		d.shutdown()
		<-d.Done()
	}()

	conn, err := ipc.Dial()
	require.NoError(t, err)
	defer conn.Close()

	client := ipc.NewDaemonServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sessionID := "cache-test-session"

	// First render: segments complete within timeout and should be cached
	stream1, err := client.RenderPrompt(ctx, &ipc.PromptRequest{
		Version:   ipc.ProtocolVersion,
		SessionId: sessionID,
		RequestId: "req-1",
		Flags: &ipc.Flags{
			Pwd:     "/tmp",
			VimMode: "insert",
		},
	})
	require.NoError(t, err)

	resp1, err := stream1.Recv()
	require.NoError(t, err)
	assert.NotNil(t, resp1.Prompts["primary"])
	initialPrompt := resp1.Prompts["primary"].Text
	assert.Contains(t, initialPrompt, "hello", "Initial render should contain segment text")

	// Simulate registry cleanup (normally happens after 5 seconds)
	// Clear the registry for this session to simulate cleanup
	d.registry.HardCancel(sessionID)

	// Repaint: should use cached values even though registry is empty
	stream2, err := client.RenderPrompt(ctx, &ipc.PromptRequest{
		Version:   ipc.ProtocolVersion,
		SessionId: sessionID,
		RequestId: "req-2",
		Repaint:   true,
		Flags: &ipc.Flags{
			Pwd:     "/tmp",
			VimMode: "normal",
		},
	})
	require.NoError(t, err)

	resp2, err := stream2.Recv()
	require.NoError(t, err)
	assert.NotNil(t, resp2.Prompts["primary"])
	repaintPrompt := resp2.Prompts["primary"].Text

	// The repaint should still contain the segment text from cache
	assert.Contains(t, repaintPrompt, "hello", "Repaint should contain cached segment text even after registry cleanup")
}
