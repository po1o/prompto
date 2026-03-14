package segments

import (
	"bufio"
	"os"
	"path/filepath"
	libruntime "runtime"
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

func TestSupportsTimeFormat(t *testing.T) {
	t.Parallel()

	assert.True(t, SupportsTimeFormat("15:04:05"))
	assert.True(t, SupportsTimeFormat("3:04 PM"))
	assert.True(t, SupportsTimeFormat("2 Jan, Monday"))
	assert.True(t, SupportsTimeFormat("DateTime"))
	assert.False(t, SupportsTimeFormat("RFC3339"))
	assert.False(t, SupportsTimeFormat("Monday <#fff>at</> 3:04 PM"))
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
			name:     "unpadded 12 hour",
			layout:   "3:04 PM",
			expected: "%-I:%M %p",
			ok:       true,
		},
		{
			name:     "unpadded day",
			layout:   "2 Jan, Monday",
			expected: "%-d %b, %A",
			ok:       true,
		},
		{
			name:     "date time zone",
			layout:   "Mon Jan _2 15:04:05 MST 2006 -0700",
			expected: "%a %b %e %H:%M:%S %Z %Y %z",
			ok:       true,
		},
		{
			name:     "kitchen supported",
			layout:   time.Kitchen,
			expected: "%-I:%M%p",
			ok:       true,
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

func TestBundledThemesUseSupportedTimeFormats(t *testing.T) {
	t.Parallel()

	_, filename, _, ok := libruntime.Caller(0)
	assert.True(t, ok)

	themeDir := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "themes"))
	entries, err := os.ReadDir(themeDir)
	assert.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		filePath := filepath.Join(themeDir, entry.Name())
		file, err := os.Open(filePath)
		assert.NoError(t, err, filePath)
		if err != nil {
			continue
		}

		func() {
			defer file.Close()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if !strings.HasPrefix(line, "time_format:") {
					continue
				}

				format := strings.TrimSpace(strings.TrimPrefix(line, "time_format:"))
				format = strings.Trim(format, `"`)
				assert.True(t, SupportsTimeFormat(format), "%s uses unsupported time_format %q", entry.Name(), format)
			}

			assert.NoError(t, scanner.Err(), filePath)
		}()
	}
}
