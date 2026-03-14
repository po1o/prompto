package shell

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var allFeatures = Tooltips | LineError | Transient | Jobs | Azure | PoshGit | FTCSMarks | PromptMark | RPrompt | CursorPositioning | Daemon

func TestPwshFeatures(t *testing.T) {
	got := allFeatures.Lines(PWSH).String("")

	want := `
$global:_promptoJobCount = $true
$global:_promptoAzure = $true
$global:_promptoPoshGit = $true
Enable-PromptoLineError
Enable-PromptoTooltips
Enable-PromptoTransientPrompt
$global:_promptoFTCSMarks = $true
Enable-PromptoDaemon`

	assert.Equal(t, want, got)
}

func TestPwshVimFeatures(t *testing.T) {
	features := VimMode | VimCursorBlink | VimCursorShape

	got := features.Lines(PWSH).String("")

	want := `
Set-PSReadLineOption -EditMode Vi; Enable-PromptoVimMode
$script:CursorBlink = $true
$script:CursorShape = $true; Set-VimModeCursorFromState`

	assert.Equal(t, want, got)
}

func TestPwshInitDecodesEscapedRenderOutput(t *testing.T) {
	assert.Contains(t, pwshInit, "function Expand-PromptoRenderText")
	assert.Contains(t, pwshInit, "$text = Expand-PromptoRenderText")
}

func TestPwshInitSupportsUnpaddedClockTokens(t *testing.T) {
	assert.Contains(t, pwshInit, "Replace('%-d', '__PROMPTO_DAY__%d__PROMPTO_END__')")
	assert.Contains(t, pwshInit, "Replace('%-I', '__PROMPTO_HOUR__%I__PROMPTO_END__')")
}

func TestQuotePwshOrElvishStr(t *testing.T) {
	tests := []struct {
		str      string
		expected string
	}{
		{str: "", expected: "''"},
		{str: `/tmp/"omp's dir"/prompto`, expected: `'/tmp/"omp''s dir"/prompto'`},
		{str: `C:/tmp\omp's dir/prompto.exe`, expected: `'C:/tmp\omp''s dir/prompto.exe'`},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.expected, quotePwshOrElvishStr(tc.str), fmt.Sprintf("quotePwshOrElvishStr: %s", tc.str))
	}
}
