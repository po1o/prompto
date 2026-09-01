package daemon

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/po1o/prompto/src/daemon/ipc"
	"github.com/po1o/prompto/src/log"
	"github.com/po1o/prompto/src/runtime"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// DialTimeout is the maximum time to wait for daemon connection.
const DialTimeout = 2 * time.Second

const (
	// startTimeout bounds how long a freshly spawned daemon is given to accept
	// connections, and startPoll how often it is tried in the meantime.
	// Measured cold starts land in the low hundreds of milliseconds: the daemon
	// loads its config before it listens, so the socket is not there the moment
	// the process is.
	startTimeout = 2 * time.Second
	startPoll    = 25 * time.Millisecond
)

// ResponseTypeComplete indicates the final response from the daemon.
const ResponseTypeComplete = "complete"

// ResponseTypeUpdate indicates a partial response with more updates to follow.
const ResponseTypeUpdate = "update"

// ResponseTypeInitial is the first response of a stream, before any segments
// are ready.
const ResponseTypeInitial = "initial"

// Stream names used as keys in PromptResponse.Prompts.
const (
	PromptPrimary    = "primary"
	PromptRight      = "right"
	PromptSecondary  = "secondary"
	PromptTransient  = "transient"
	PromptRTransient = "rtransient"
)

// ResponseCallback is called for each response from the daemon.
// Return false to stop receiving responses.
type ResponseCallback func(*ipc.PromptResponse) bool

// Client handles communication with the daemon.
type Client struct {
	conn   *grpc.ClientConn
	client ipc.DaemonServiceClient
}

// NewClient creates a new daemon client.
// Returns an error if the daemon is not running.
func NewClient() (*Client, error) {
	conn, err := ipc.Dial()
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	// gRPC uses lazy connection, so we need to explicitly connect and verify.
	// Connect() initiates connection, then we wait for Ready state.
	ctx, cancel := context.WithTimeout(context.Background(), DialTimeout)
	defer cancel()

	conn.Connect()

	// Wait for connection to become ready
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			break
		}
		if state == connectivity.TransientFailure || state == connectivity.Shutdown {
			conn.Close()
			return nil, fmt.Errorf("daemon not available: connection state %s", state)
		}
		// Wait for state change or timeout
		if !conn.WaitForStateChange(ctx, state) {
			conn.Close()
			return nil, fmt.Errorf("connection timeout: daemon not responding")
		}
	}

	return &Client{
		conn:   conn,
		client: ipc.NewDaemonServiceClient(conn),
	}, nil
}

// ConnectOrStart attempts to connect to the daemon. If that fails it kills any
// stale daemon, calls startFunc to start a fresh one, and waits for it to come
// up.
func ConnectOrStart(startFunc func() error) (*Client, error) {
	client, err := NewClient()
	if err == nil {
		return client, nil
	}

	// Force kill ANY existing daemon/lock, for a clean slate to start from.
	_ = KillDaemon()

	if err := startFunc(); err != nil {
		return nil, fmt.Errorf("failed to start daemon: %w", err)
	}

	return waitForDaemon()
}

// waitForDaemon retries until the daemon accepts a connection. A connect to a
// socket that is not bound yet fails at once rather than blocking, so a single
// attempt after a fixed sleep gives the daemon only that sleep to get ready —
// and it needs longer than any sleep short enough to keep a prompt responsive.
// Retrying returns as soon as the daemon is up instead of always paying the
// wait, and it lets a shell that lost the race to spawn the daemon connect to
// the winner's rather than fail.
func waitForDaemon() (*Client, error) {
	deadline := time.Now().Add(startTimeout)

	for {
		client, err := NewClient()
		if err == nil {
			return client, nil
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("daemon did not accept connections within %s: %w", startTimeout, err)
		}

		time.Sleep(startPoll)
	}
}

// Close closes the client connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// RenderPrompt sends a prompt render request to the daemon and streams responses.
//
// The daemon returns responses incrementally:
//   - After async_timeout (100ms): partial prompt with fast segments + cached values for slow ones
//   - As slow segments complete: streamed updates for shell to repaint
//   - Final "complete" response when all segments are done
//
// The callback is invoked for each response. Return false from callback to stop receiving.
// The requestID is automatically generated and used to filter stale responses.
//
// If repaint is true, the daemon performs a "soft cancel" - existing computations continue
// and can be reused (used for vim mode toggles). If false, "hard cancel" aborts computations.
func (c *Client) RenderPrompt(ctx context.Context, flags *runtime.Flags, pid int, sessionID string, env map[string]string, repaint bool, callback ResponseCallback) error {
	requestID := uuid.NewString()

	// The session is the shell, identified by its pid: the daemon watches that
	// process to know when to reap the session. Every shell integration passes
	// one; a hand-run `prompto render` does not, so fall back to the parent,
	// which is the shell that invoked us.
	if sessionID == "" {
		sessionID = sessionIDForPID(pid)
	}

	req := &ipc.PromptRequest{
		Version:   ipc.ProtocolVersion,
		SessionId: sessionID,
		RequestId: requestID,
		Pid:       int32(pid),
		Env:       env,
		Flags:     ipc.FlagsToProto(flags),
		Repaint:   repaint,
	}

	log.Debugf("Sending prompt request: session=%s, request=%s, type=%s", sessionID, requestID, flags.Type)

	// NOTE: c.client.RenderPrompt is the gRPC-generated method on ipc.DaemonServiceClient,
	// not a recursive call to this method. The gRPC client is stored in c.client.
	stream, err := c.client.RenderPrompt(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send render request: %w", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("stream error: %w", err)
		}

		// Filter stale responses from previous requests (different request ID)
		if resp.RequestId != requestID {
			log.Debugf("Ignoring stale response: got %s, expected %s", resp.RequestId, requestID)
			continue
		}

		if resp.Error != "" {
			return fmt.Errorf("daemon error: %s", resp.Error)
		}

		if !callback(resp) {
			return nil
		}

		if resp.Type == ResponseTypeComplete {
			return nil
		}
	}
}

// RenderPromptSync sends a prompt render request and waits for the complete response.
// This is a convenience wrapper for cases that don't need incremental updates.
func (c *Client) RenderPromptSync(ctx context.Context, flags *runtime.Flags, pid int, sessionID string, env map[string]string, repaint bool) (*ipc.PromptResponse, error) {
	var finalResponse *ipc.PromptResponse

	err := c.RenderPrompt(ctx, flags, pid, sessionID, env, repaint, func(resp *ipc.PromptResponse) bool {
		finalResponse = resp
		return resp.Type != ResponseTypeComplete
	})
	if err != nil {
		return nil, err
	}

	return finalResponse, nil
}

// ToggleSegment toggles segments in the daemon.
func (c *Client) ToggleSegment(ctx context.Context, pid int, segments []string) error {
	req := &ipc.ToggleSegmentRequest{
		SessionId: sessionIDForPID(pid),
		Segments:  segments,
	}

	resp, err := c.client.ToggleSegment(ctx, req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("%s", resp.Error)
	}

	return nil
}

// CacheClear clears all daemon cache entries.
func (c *Client) CacheClear(ctx context.Context) error {
	resp, err := c.client.CacheClear(ctx, &ipc.CacheClearRequest{})
	if err != nil {
		return err
	}
	if !resp.Success && resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

// CacheSetTTL sets the default cache TTL (in days).
func (c *Client) CacheSetTTL(ctx context.Context, days int) error {
	_, err := c.client.CacheSetTTL(ctx, &ipc.CacheSetTTLRequest{Days: int32(days)})
	return err
}

// CacheGetTTL gets the current default cache TTL (in days).
func (c *Client) CacheGetTTL(ctx context.Context) (int, error) {
	resp, err := c.client.CacheGetTTL(ctx, &ipc.CacheGetTTLRequest{})
	if err != nil {
		return 0, err
	}
	return int(resp.Days), nil
}

// SetLogging enables or disables file logging on the running daemon.
func (c *Client) SetLogging(ctx context.Context, path string) error {
	resp, err := c.client.SetLogging(ctx, &ipc.SetLoggingRequest{Path: path})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

// sessionIDForPID names the session a request belongs to. The daemon parses it
// back into a pid to watch the process, so it has to stay a number.
func sessionIDForPID(pid int) string {
	if pid > 0 {
		return fmt.Sprint(pid)
	}

	return fmt.Sprint(os.Getppid())
}

// IsRunning checks if the daemon is currently running.
func IsRunning() bool {
	client, err := NewClient()
	if err != nil {
		return false
	}
	client.Close()
	return true
}

// PromptResult contains the rendered prompts from a daemon response.
type PromptResult struct {
	Primary    string
	Right      string
	Secondary  string
	Transient  string
	RTransient string
	Debug      string
	Tooltip    string
	Valid      string
	Error      string
}

// ExtractPrompts converts a PromptResponse into a PromptResult.
func ExtractPrompts(resp *ipc.PromptResponse) *PromptResult {
	result := &PromptResult{}
	if resp == nil || resp.Prompts == nil {
		return result
	}

	for _, field := range []struct {
		dst *string
		key string
	}{
		{&result.Primary, PromptPrimary},
		{&result.Right, PromptRight},
		{&result.Secondary, PromptSecondary},
		{&result.Transient, PromptTransient},
		{&result.RTransient, PromptRTransient},
		{&result.Debug, "debug"},
		{&result.Tooltip, "tooltip"},
		{&result.Valid, "valid"},
		{&result.Error, "error"},
	} {
		if p, ok := resp.Prompts[field.key]; ok {
			*field.dst = p.Text
		}
	}

	return result
}
