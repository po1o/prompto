package shell

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDaemonScriptsIncludePIDAndVimModeSupport(t *testing.T) {
	testCases := []struct {
		name   string
		script string
	}{
		{name: "bash", script: bashInit},
		{name: "zsh", script: zshInit},
		{name: "fish", script: fishInit},
		{name: "pwsh", script: pwshInit},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, strings.Contains(tc.script, "render"), "expected render command in daemon script")
			assert.True(t, strings.Contains(tc.script, "--pid"), "expected --pid in daemon render command")
			assert.True(t, strings.Contains(tc.script, "vim"), "expected vim mode handling in daemon script")
		})
	}
}
