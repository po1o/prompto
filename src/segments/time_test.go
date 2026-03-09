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
	lastDate := time.Now()
	cases := []struct {
		Case            string
		ExpectedString  string
		Template        string
		ExpectedEnabled bool
	}{
		{
			Case:            "no template",
			Template:        "",
			ExpectedString:  lastDate.Format("15:04:05"),
			ExpectedEnabled: true,
		},
		{
			Case:            "time only",
			Template:        "{{.LastDate | date \"15:04:05\"}}",
			ExpectedString:  lastDate.Format("15:04:05"),
			ExpectedEnabled: true,
		},
		{
			Case:            "lowercase",
			Template:        "{{.LastDate | date \"January 02, 2006 15:04:05\" | lower }}",
			ExpectedString:  strings.ToLower(lastDate.Format("January 02, 2006 15:04:05")),
			ExpectedEnabled: true,
		},
	}

	for _, tc := range cases {
		env := new(mock.Environment)
		env.On("Shell").Return(shell.FISH)

		tempus := &Time{
			LastDate: lastDate,
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

func TestTimeCurrentDateDisplayForZsh(t *testing.T) {
	env := new(mock.Environment)
	env.On("Shell").Return(shell.ZSH)

	timeSegment := &Time{}
	timeSegment.Init(options.Map{
		TimeFormat: "15:04",
	}, env)

	assert.True(t, timeSegment.Enabled())
	assert.Equal(t, "%D{%H:%M}", timeSegment.CurrentDate)
}

func TestTimeCurrentDateDisplayForBash(t *testing.T) {
	env := new(mock.Environment)
	env.On("Shell").Return(shell.BASH)

	timeSegment := &Time{}
	timeSegment.Init(options.Map{
		TimeFormat: "15:04",
	}, env)

	assert.True(t, timeSegment.Enabled())
	assert.Equal(t, "\\D{%H:%M}", timeSegment.CurrentDate)
}

func TestTimeCurrentDateDisplayPlaceholderForFishAndPwsh(t *testing.T) {
	lastDate := time.Date(2026, 3, 7, 15, 4, 5, 0, time.UTC)
	shellsWithPlaceholder := []string{shell.FISH, shell.PWSH}

	for _, shellName := range shellsWithPlaceholder {
		env := new(mock.Environment)
		env.On("Shell").Return(shellName)

		timeSegment := &Time{LastDate: lastDate}
		timeSegment.Init(options.Map{
			TimeFormat: "15:04",
		}, env)

		assert.True(t, timeSegment.Enabled())
		assert.Equal(t, "__PROMPTO_CLOCK{%H:%M}__", timeSegment.CurrentDate)
	}
}

func TestGoLayoutToStrftime(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		layout   string
		expected string
		ok       bool
	}{
		{
			name:     "simple time",
			layout:   "15:04:05",
			expected: "%H:%M:%S",
			ok:       true,
		},
		{
			name:     "date time zone",
			layout:   "Mon Jan _2 15:04:05 MST 2006 -0700",
			expected: "%a %b %e %H:%M:%S %Z %Y %z",
			ok:       true,
		},
		{
			name:   "kitchen unsupported",
			layout: time.Kitchen,
			ok:     false,
		},
		{
			name:   "rfc3339 unsupported",
			layout: time.RFC3339,
			ok:     false,
		},
		{
			name:   "fractional seconds unsupported",
			layout: time.StampMilli,
			ok:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, ok := goLayoutToStrftime(tc.layout)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestTimeCurrentDateFallsBackToRenderedValueWhenLayoutIsNotTranslatable(t *testing.T) {
	t.Parallel()

	lastDate := time.Date(2026, 3, 7, 15, 4, 5, 0, time.UTC)
	env := new(mock.Environment)
	env.On("Shell").Return(shell.ZSH)

	timeSegment := &Time{LastDate: lastDate}
	timeSegment.Init(options.Map{
		TimeFormat: "Kitchen",
	}, env)

	assert.True(t, timeSegment.Enabled())
	assert.Equal(t, lastDate.Format(time.Kitchen), timeSegment.CurrentDate)
}

func TestTimeCurrentDateUsesTimeFormatConstantWhenTranslatable(t *testing.T) {
	t.Parallel()

	env := new(mock.Environment)
	env.On("Shell").Return(shell.ZSH)

	timeSegment := &Time{}
	timeSegment.Init(options.Map{
		TimeFormat: "DateTime",
	}, env)

	assert.True(t, timeSegment.Enabled())
	assert.Equal(t, "%D{%Y-%m-%d %H:%M:%S}", timeSegment.CurrentDate)
}

func TestTimeLastDateKeepsRenderedTimestamp(t *testing.T) {
	t.Parallel()

	lastDate := time.Date(2026, 3, 7, 15, 4, 5, 0, time.UTC)
	env := new(mock.Environment)
	env.On("Shell").Return(shell.ZSH)

	timeSegment := &Time{LastDate: lastDate}
	timeSegment.Init(options.Map{
		TimeFormat: "15:04",
	}, env)

	assert.True(t, timeSegment.Enabled())
	assert.Equal(t, lastDate, timeSegment.LastDate)
}
