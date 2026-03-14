package segments

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/po1o/prompto/src/segments/options"
	"github.com/po1o/prompto/src/shell"
)

type Time struct {
	Base

	LastDate    time.Time
	Format      string
	CurrentDate string
}

const (
	// TimeFormat uses the reference time Mon Jan 2 15:04:05 MST 2006 to show the pattern with which to format the current time
	TimeFormat options.Option = "time_format"
)

const defaultTimeFormat = "15:04:05"

func (t *Time) Template() string {
	return "{{ .LastDate | date .Format }}"
}

func (t *Time) Enabled() bool {
	formatInput := t.options.String(TimeFormat, defaultTimeFormat)
	t.Format = ResolveTimeFormat(formatInput)

	if t.LastDate.IsZero() {
		t.LastDate = time.Now()
	}
	t.CurrentDate = t.LastDate.Format(t.Format)

	if currentDate, ok := t.currentDateDisplay(); ok {
		t.CurrentDate = currentDate
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

// ResolveTimeFormat returns the resolved Go time layout when the input matches a supported named format.
// Otherwise it returns the input unchanged.
func ResolveTimeFormat(format string) string {
	if timeFormat, exists := timeFormatLookup[format]; exists {
		return timeFormat
	}
	return format
}

func (t *Time) currentDateDisplay() (string, bool) {
	if t.env == nil {
		return "", false
	}

	strftime, ok := goLayoutToStrftime(t.Format)
	if !ok {
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

type currentDateToken struct {
	goLayout string
	strftime string
}

var currentDateTokens = []currentDateToken{
	{goLayout: "Monday", strftime: "%A"},
	{goLayout: "January", strftime: "%B"},
	{goLayout: "2006", strftime: "%Y"},
	{goLayout: "-0700", strftime: "%z"},
	{goLayout: "Mon", strftime: "%a"},
	{goLayout: "Jan", strftime: "%b"},
	{goLayout: "MST", strftime: "%Z"},
	{goLayout: "_2", strftime: "%e"},
	{goLayout: "15", strftime: "%H"},
	{goLayout: "03", strftime: "%I"},
	{goLayout: "3", strftime: "%-I"},
	{goLayout: "PM", strftime: "%p"},
	{goLayout: "02", strftime: "%d"},
	{goLayout: "2", strftime: "%-d"},
	{goLayout: "01", strftime: "%m"},
	{goLayout: "06", strftime: "%y"},
	{goLayout: "05", strftime: "%S"},
	{goLayout: "04", strftime: "%M"},
}

var unsupportedCurrentDateTokens = []string{
	"-07:00",
	"Z07:00",
	"Z0700",
	".000000000",
	".999999999",
	".000000",
	".999999",
	".000",
	".999",
	"-07",
	"pm",
	"1",
	"4",
	"5",
}

func SupportsTimeFormat(format string) bool {
	format = ResolveTimeFormat(format)

	if strings.ContainsAny(format, "<>") {
		return false
	}

	_, ok := goLayoutToStrftime(format)
	return ok
}

func goLayoutToStrftime(layout string) (string, bool) {
	if layout == "" {
		return "", false
	}

	var builder strings.Builder

	for len(layout) > 0 {
		matched := false

		for _, token := range currentDateTokens {
			if !strings.HasPrefix(layout, token.goLayout) {
				continue
			}

			builder.WriteString(token.strftime)
			layout = layout[len(token.goLayout):]
			matched = true
			break
		}

		if matched {
			continue
		}

		for _, token := range unsupportedCurrentDateTokens {
			if !strings.HasPrefix(layout, token) {
				continue
			}

			return "", false
		}

		r, size := utf8.DecodeRuneInString(layout)
		builder.WriteRune(r)
		layout = layout[size:]
	}

	return builder.String(), true
}
