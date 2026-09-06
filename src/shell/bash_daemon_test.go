package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// bashDriver stands in for ble.sh. job.start takes a command *string* and
// parses it, so the driver captures that string and then evaluates it exactly
// as ble.sh would, reporting the argv the render binary would really receive.
//
// Two constraints on ::SETUP::. It runs after the script is sourced, so it can
// only influence state the script reads when _prompto_daemon_render is called,
// not while sourcing. And nothing here fails on a bad setup line — a misspelt
// variable is silently inert and the driver still exits zero — so a subtest has
// to assert what it expects to see, never only the absence of something.
const bashDriver = `
function ble/util/job.start() {
  printf 'CAPTURED_COMMAND %s\n' "$1"
  printf 'CAPTURED_CALLBACK %s\n' "$2"
  __captured_command=$1
}

source ::SCRIPT::

_prompto_status=0
_prompto_pipestatus=(0 0)
_prompto_no_status=true
_prompto_execution_time=-1
_prompto_stack_count=0

::SETUP::

_prompto_daemon_render ::REPAINT::

# Evaluate it the way ble.sh does. The fake executable prints one argument per
# line, so the test sees what the render binary would actually be handed.
eval "$__captured_command"
printf 'EVAL_STATUS %s\n' "$?"
`

// fakeRenderBinary prints its arguments one per line so the test can assert on
// the argv that survived the round trip through the command string.
const fakeRenderBinary = `#!/usr/bin/env bash
for arg in "$@"; do printf 'ARG %s\n' "$arg"; done
`

// TestBashDaemonRenderSurvivesBeingReparsed executes the command prompto.bash
// hands to ble.sh, rather than pattern-matching the script.
//
// ble/util/job.start parses that string, so every value interpolated into it
// has to survive being read as shell source. Two of them did not:
// $BASH_VERSION always contains "(1)", which is a syntax error that aborts the
// whole command — so the render never ran at all and the prompt kept the
// placeholders — and $PWD was expanded twice, making a directory named
// '$(...)' execute on every prompt.
func TestBashDaemonRenderSurvivesBeingReparsed(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not installed")
	}

	t.Run("the command parses and carries every argument", func(t *testing.T) {
		output := runBashDriver(t, t.TempDir(), "", "")

		require.Contains(t, output, "CAPTURED_CALLBACK _prompto_daemon_job",
			"the output-consuming callback must be job.start's second argument")
		requireCommandParsed(t, output)

		require.Contains(t, output, "ARG render")
		require.Contains(t, output, "ARG --shell=bash")
		require.Contains(t, output, "ARG --escape=false")

		// The value that used to break it, intact and in one piece.
		require.Contains(t, output, "ARG --shell-version="+bashVersion(t))
	})

	t.Run("a repaint flag reaches the binary", func(t *testing.T) {
		output := runBashDriver(t, t.TempDir(), "--repaint", "")

		requireCommandParsed(t, output)
		require.Contains(t, output, "ARG --repaint")
	})

	t.Run("the conditional arguments survive too", func(t *testing.T) {
		// Both are appended only when configured, so the plain path above never
		// reaches them — and a config path is user-supplied, so it carries the
		// same hazards as $PWD.
		setup := `_prompto_config='/tmp/my cfg$(id -u)/a b.omp.yaml'
_prompto_vim_mode=1
BLE_VERSION=0.4
_ble_decode_keymap=vi_nmap`

		output := runBashDriver(t, t.TempDir(), "", setup)

		requireCommandParsed(t, output)
		require.Contains(t, output, "ARG --config=/tmp/my cfg$(id -u)/a b.omp.yaml",
			"a config path must arrive whole and unexpanded")
		require.Contains(t, output, "ARG --vim-mode=normal")
	})

	t.Run("a directory name is not executed", func(t *testing.T) {
		// A name that runs a command if the value is ever expanded again.
		hostile := filepath.Join(t.TempDir(), "inj$(id -u)dir")
		require.NoError(t, os.MkdirAll(hostile, 0o755))

		output := runBashDriver(t, hostile, "", "")

		requireCommandParsed(t, output)
		require.Contains(t, output, "ARG --pwd="+hostile,
			"the directory name must pass through literally")
		require.NotContains(t, output, "inj0dir")
		require.NotContains(t, output, "inj1000dir")
	})

	t.Run("a directory name with spaces stays one argument", func(t *testing.T) {
		spaced := filepath.Join(t.TempDir(), "two words")
		require.NoError(t, os.MkdirAll(spaced, 0o755))

		output := runBashDriver(t, spaced, "", "")

		requireCommandParsed(t, output)
		require.Contains(t, output, "ARG --pwd="+spaced)
	})
}

// requireCommandParsed is the cause behind every other assertion here: a
// command that does not parse produces no arguments at all, and a bare missing
// ARG says much less about why.
func requireCommandParsed(t *testing.T, output string) {
	t.Helper()

	require.Contains(t, output, "EVAL_STATUS 0",
		"the command must parse; a syntax error means the render never runs at all")
}

func runBashDriver(t *testing.T, workingDir, repaint, setup string) string {
	t.Helper()

	dir := t.TempDir()

	executable := filepath.Join(dir, "prompto")
	require.NoError(t, os.WriteFile(executable, []byte(fakeRenderBinary), 0o700))

	script := filepath.Join(dir, "prompto.bash")
	integration := strings.NewReplacer(
		"::PROMPTO::", "'"+executable+"'",
		"::CONFIG::", "''",
	).Replace(bashInit)
	require.NoError(t, os.WriteFile(script, []byte(integration), 0o600))

	driver := filepath.Join(dir, "driver.bash")
	body := strings.NewReplacer(
		"::SCRIPT::", "'"+script+"'",
		"::REPAINT::", repaint,
		"::SETUP::", setup,
	).Replace(bashDriver)
	require.NoError(t, os.WriteFile(driver, []byte(body), 0o600))

	command := exec.CommandContext(t.Context(), "bash", driver)
	command.Dir = workingDir

	output, err := command.CombinedOutput()
	require.NoError(t, err, "driver failed:\n%s", output)

	return string(output)
}

func bashVersion(t *testing.T) string {
	t.Helper()

	command := exec.CommandContext(t.Context(), "bash", "-c", "printf %s \"$BASH_VERSION\"")
	output, err := command.Output()
	require.NoError(t, err)

	return string(output)
}
