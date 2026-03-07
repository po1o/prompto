package segments

import (
	"strings"
	"testing"
	"time"

	"github.com/po1o/prompto/src/runtime/mock"
	"github.com/po1o/prompto/src/segments/options"
	"github.com/po1o/prompto/src/shell"

	"github.com/stretchr/testify/assert"
)

func TestTimeSegmentTemplate(t *testing.T) {
	// set date for unit test
	currentDate := time.Now()
	cases := []struct {
		Case            string
		ExpectedString  string
		Template        string
		ExpectedEnabled bool
	}{
		{
			Case:            "no template",
			Template:        "",
			ExpectedString:  currentDate.Format("15:04:05"),
			ExpectedEnabled: true,
		},
		{
			Case:            "time only",
			Template:        "{{.CurrentDate | date \"15:04:05\"}}",
			ExpectedString:  currentDate.Format("15:04:05"),
			ExpectedEnabled: true,
		},
		{
			Case:            "lowercase",
			Template:        "{{.CurrentDate | date \"January 02, 2006 15:04:05\" | lower }}",
			ExpectedString:  strings.ToLower(currentDate.Format("January 02, 2006 15:04:05")),
			ExpectedEnabled: true,
		},
	}

	for _, tc := range cases {
		env := new(mock.Environment)
		env.On("Shell").Return(shell.FISH)

		tempus := &Time{
			CurrentDate: currentDate,
		}
		tempus.Init(options.Map{}, env)

		assert.Equal(t, tc.ExpectedEnabled, tempus.Enabled())
		if tc.Template == "" {
			tc.Template = tempus.Template()
		}
		if tc.ExpectedEnabled {
			assert.Equal(t, tc.ExpectedString, renderTemplate(env, tc.Template, tempus), tc.Case)
		}
	}
}

func TestTimeShellClockDisplayForZsh(t *testing.T) {
	env := new(mock.Environment)
	env.On("Shell").Return(shell.ZSH)

	timeSegment := &Time{}
	timeSegment.Init(options.Map{
		TimeFormat: "15:04",
	}, env)

	assert.True(t, timeSegment.Enabled())
	assert.Equal(t, "%D{%H:%M}", timeSegment.ShellClock)
}

func TestTimeShellClockDisplayForBash(t *testing.T) {
	env := new(mock.Environment)
	env.On("Shell").Return(shell.BASH)

	timeSegment := &Time{}
	timeSegment.Init(options.Map{
		TimeFormat: "15:04",
	}, env)

	assert.True(t, timeSegment.Enabled())
	assert.Equal(t, "\\D{%H:%M}", timeSegment.ShellClock)
}

func TestTimeShellClockDisplayPlaceholderForFishAndPwsh(t *testing.T) {
	currentDate := time.Date(2026, 3, 7, 15, 4, 5, 0, time.UTC)
	shellsWithPlaceholder := []string{shell.FISH, shell.PWSH}

	for _, shellName := range shellsWithPlaceholder {
		env := new(mock.Environment)
		env.On("Shell").Return(shellName)

		timeSegment := &Time{CurrentDate: currentDate}
		timeSegment.Init(options.Map{
			TimeFormat: "15:04",
		}, env)

		assert.True(t, timeSegment.Enabled())
		assert.Equal(t, "__PROMPTO_CLOCK{%H:%M}__", timeSegment.ShellClock)
	}
}
