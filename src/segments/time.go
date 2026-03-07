package segments

import (
	"strings"
	"time"

	"github.com/po1o/prompto/src/segments/options"
	"github.com/po1o/prompto/src/shell"
)

type Time struct {
	Base

	CurrentDate time.Time
	Format      string
	ShellClock  string
}

const (
	// TimeFormat uses the reference time Mon Jan 2 15:04:05 MST 2006 to show the pattern with which to format the current time
	TimeFormat options.Option = "time_format"
)

func (t *Time) Template() string {
	return "{{ .CurrentDate | date .Format }}"
}

func (t *Time) Enabled() bool {
	formatInput := t.options.String(TimeFormat, "15:04:05")
	t.Format = t.getTimeFormat(formatInput)

	if t.CurrentDate.IsZero() {
		t.CurrentDate = time.Now()
	}
	t.ShellClock = t.CurrentDate.Format(t.Format)

	if shellClock, ok := t.shellClockDisplay(); ok {
		t.ShellClock = shellClock
	}

	return true
}

var timeFormatLookup = map[string]string{
	// Maps string names to their corresponding time package constants
	"Layout":      time.Layout,      // "01/02 03:04:05PM '06 -0700"
	"ANSIC":       time.ANSIC,       // "Mon Jan _2 15:04:05 2006"
	"UnixDate":    time.UnixDate,    // "Mon Jan _2 15:04:05 MST 2006"
	"RubyDate":    time.RubyDate,    // "Mon Jan 02 15:04:05 -0700 2006"
	"RFC822":      time.RFC822,      // "02 Jan 06 15:04 MST"
	"RFC822Z":     time.RFC822Z,     // "02 Jan 06 15:04 -0700"
	"RFC850":      time.RFC850,      // "Monday, 02-Jan-06 15:04:05 MST"
	"RFC1123":     time.RFC1123,     // "Mon, 02 Jan 2006 15:04:05 MST"
	"RFC1123Z":    time.RFC1123Z,    // "Mon, 02 Jan 2006 15:04:05 -0700"
	"RFC3339":     time.RFC3339,     // "2006-01-02T15:04:05Z07:00"
	"RFC3339Nano": time.RFC3339Nano, // "2006-01-02T15:04:05.999999999Z07:00"
	"Kitchen":     time.Kitchen,     // "3:04PM"
	"Stamp":       time.Stamp,       // "Jan _2 15:04:05"
	"StampMilli":  time.StampMilli,  // "Jan _2 15:04:05.000"
	"StampMicro":  time.StampMicro,  // "Jan _2 15:04:05.000000"
	"StampNano":   time.StampNano,   // "Jan _2 15:04:05.000000000"
	"DateTime":    time.DateTime,    // "2006-01-02 15:04:05"
	"DateOnly":    time.DateOnly,    // "2006-01-02"
	"TimeOnly":    time.TimeOnly,    // "15:04:05"
}

// getTimeFormat returns the time format constant if the input matches a known format name,
// otherwise returns the input unchanged
func (t *Time) getTimeFormat(format string) string {
	if timeFormat, exists := timeFormatLookup[format]; exists {
		return timeFormat
	}
	return format
}

func (t *Time) shellClockDisplay() (string, bool) {
	if t.env == nil {
		return "", false
	}

	strftime := goLayoutToStrftime(t.Format)
	if strftime == "" {
		return "", false
	}

	switch t.env.Shell() {
	case shell.ZSH:
		return "%D{" + strftime + "}", true
	case shell.BASH:
		return "\\D{" + strftime + "}", true
	case shell.FISH, shell.PWSH:
		return "__PROMPTO_CLOCK{" + strftime + "}__", true
	default:
		return "", false
	}
}

func goLayoutToStrftime(layout string) string {
	if layout == "" {
		return ""
	}

	replacer := strings.NewReplacer(
		"Monday", "%A",
		"January", "%B",
		"Mon", "%a",
		"Jan", "%b",
		"2006", "%Y",
		"06", "%y",
		"15", "%H",
		"03", "%I",
		"04", "%M",
		"05", "%S",
		"PM", "%p",
		"pm", "%p",
		"02", "%d",
		"01", "%m",
		"_2", "%e",
		"2", "%-d",
		"1", "%-m",
		"MST", "%Z",
		"-0700", "%z",
		"-07", "%z",
	)

	return replacer.Replace(layout)
}
