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
		expectedStart   string
		expectedPIDFlag string
	}{
		{
			name:            "bash",
			script:          bashInit,
			expectedStart:   "daemon start --config=\"$_prompto_config\"",
			expectedPIDFlag: "--pid=$$",
		},
		{
			name:            "zsh",
			script:          zshInit,
			expectedStart:   "daemon start --config=$_prompto_config",
			expectedPIDFlag: "--pid=$$",
		},
		{
			name:            "fish",
			script:          fishInit,
			expectedStart:   "daemon start --config=$_prompto_config",
			expectedPIDFlag: "--pid=$parent_pid",
		},
		{
			name:            "pwsh",
			script:          pwshInit,
			expectedStart:   "-ArgumentList \"daemon\", \"start\", \"--config\"",
			expectedPIDFlag: "--pid=$PID",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, strings.Contains(tc.script, "render"), "expected render command in daemon script")
			assert.True(t, strings.Contains(tc.script, tc.expectedStart), "expected daemon start command")
			assert.True(t, strings.Contains(tc.script, tc.expectedPIDFlag), "expected shell-specific PID forwarding")
			assert.True(t, strings.Contains(tc.script, "vim"), "expected vim mode handling in daemon script")
		})
	}
}
