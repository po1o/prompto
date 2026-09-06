package shell

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDaemonScriptsWireRepaintAndModeDetection locks the contract every
// shell script must satisfy for the daemon's Soft-Cancel path to work:
// each script (1) appends --repaint to its render call, and (2) contains
// the shell-specific mode-detection idiom that triggers the repaint.
// See src/daemon/ARCHITECTURE.md ("The cancel model") and
// docs/maintainers/daemon-shells.md.
func TestDaemonScriptsWireRepaintAndModeDetection(t *testing.T) {
	testCases := []struct {
		name              string
		script            string
		modeDetectionHook string
	}{
		{
			name:              "bash",
			script:            bashInit,
			modeDetectionHook: "_prompto_ble_keymap_change",
		},
		{
			name:              "fish",
			script:            fishInit,
			modeDetectionHook: "_prompto_on_bind_mode_change --on-variable fish_bind_mode",
		},
		{
			name:              "pwsh",
			script:            pwshInit,
			modeDetectionHook: "-ViMode Command",
		},
		{
			name:              "zsh",
			script:            zshInit,
			modeDetectionHook: "_prompto_zle-keymap-select",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, strings.Contains(tc.script, "--repaint"),
				"daemon script must append --repaint for vim-mode toggles (Soft-Cancel path)")
			assert.True(t, strings.Contains(tc.script, tc.modeDetectionHook),
				"daemon script must contain shell-specific mode-detection hook %q", tc.modeDetectionHook)
		})
	}
}

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

// TestFishDaemonReaderPublishesBatchesAtomically locks the handoff between the
// background reader and the USR1 handler. The handler reads the prompt file at
// a moment the reader does not control, so the reader must publish whole
// batches by rename and must not truncate or delete the file it reads.
func TestFishDaemonReaderPublishesBatchesAtomically(t *testing.T) {
	assert.True(t, strings.Contains(fishInit, "mv -f $batch_file $prompt_file"),
		"fish reader must publish each batch by rename so the handler never reads a half-written one")

	assert.False(t, strings.Contains(fishInit, "echo -n \"\" > $prompt_file"),
		"fish reader must not truncate the prompt file in place; a handler reading between the signal and the next line loses the batch")

	assert.False(t, strings.Contains(fishInit, "rm -f $prompt_file"),
		"fish reader must not delete the prompt file it just published; a signal still in flight would find nothing and leave the prompt pending")

	publish := strings.Index(fishInit, "mv -f $batch_file $prompt_file")
	signal := strings.Index(fishInit, "kill -USR1 $parent_pid")
	assert.True(t, publish != -1 && signal != -1 && publish < signal,
		"fish reader must publish the batch before signalling the parent")
}
