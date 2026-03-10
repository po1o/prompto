package shell

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDaemonScriptsIncludePIDAndVimModeSupport(t *testing.T) {
	testCases := []struct {
		name            string
		script          string
		expectedPIDFlag string
	}{
		{
			name:            "bash",
			script:          bashInit,
			expectedPIDFlag: "--pid=$$",
		},
		{
			name:            "zsh",
			script:          zshInit,
			expectedPIDFlag: "--pid=$$",
		},
		{
			name:            "fish",
			script:          fishInit,
			expectedPIDFlag: "--pid=$parent_pid",
		},
		{
			name:            "pwsh",
			script:          pwshInit,
			expectedPIDFlag: "--pid=$PID",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, strings.Contains(tc.script, "render"), "expected render command in daemon script")
			assert.True(t, strings.Contains(tc.script, tc.expectedPIDFlag), "expected shell-specific PID forwarding")
			assert.True(t, strings.Contains(tc.script, "vim"), "expected vim mode handling in daemon script")
			assert.NotContains(t, tc.script, "daemon start", "expected daemon renders to rely on render auto-start")
		})
	}
}
