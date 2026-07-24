//go:build !linux && !darwin && !freebsd

package daemon

import "context"

func waitForProcessExit(ctx context.Context, pid int) {
	pollForProcessExit(ctx, pid)
}
