//go:build !linux && !darwin && !windows && !freebsd

package daemon

import "context"

func waitForProcessExit(ctx context.Context, pid int) {
	pollForProcessExit(ctx, pid)
}
