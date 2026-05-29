package daemon

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCancelKindForRepaint(t *testing.T) {
	require.Equal(t, CancelSoft, CancelKindForRepaint(true), "repaint must map to soft cancel")
	require.Equal(t, CancelHard, CancelKindForRepaint(false), "non-repaint must map to hard cancel")
}

func TestCancelKindRepaintPredicate(t *testing.T) {
	require.True(t, CancelSoft.Repaint())
	require.False(t, CancelHard.Repaint())
}

func TestCancelKindString(t *testing.T) {
	require.Equal(t, "hard", CancelHard.String())
	require.Equal(t, "soft", CancelSoft.String())
	require.Equal(t, "unknown", CancelKind(99).String())
}

func TestWithCancelKindPreservesContextAndKind(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rc := WithCancelKind(ctx, CancelSoft)
	require.Equal(t, CancelSoft, rc.Kind)
	require.NoError(t, rc.Err())

	// The embedded context still drives cancellation.
	cancel()
	require.Error(t, rc.Err())
}
