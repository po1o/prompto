package prompt

import (
	"testing"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/color"
	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/maps"
	"github.com/po1o/prompto/src/runtime"
	"github.com/po1o/prompto/src/shell"
	"github.com/po1o/prompto/src/template"
	"github.com/po1o/prompto/src/terminal"

	"github.com/stretchr/testify/require"
)

// These tests are about color, so unlike newLayoutTestEngine they render with
// escape sequences intact: whether a separator carries a background is the
// whole question.
func newSeparatorTestEngine(t *testing.T, layout *config.LayoutConfig) *Engine {
	t.Helper()

	flags := &runtime.Flags{Shell: shell.GENERIC, TerminalWidth: 80}

	env := &runtime.Terminal{}
	env.Init(flags)

	template.Cache = &cache.Template{
		SimpleTemplate: cache.SimpleTemplate{Shell: shell.GENERIC},
		Segments:       maps.NewConcurrent[any](),
	}
	template.Init(env, nil, nil)

	original := terminal.Plain
	terminal.Init(shell.GENERIC)
	terminal.Colors = &color.Defaults{}
	terminal.Plain = false

	t.Cleanup(func() {
		terminal.Plain = original
	})

	return &Engine{Env: env, Config: &config.Config{}, LayoutConfig: layout}
}

const (
	red  = "#ff0000"
	blue = "#0000ff"

	onRed   = "\x1b[48;2;255;0;0m"
	onBlue  = "\x1b[48;2;0;0;255m"
	inRed   = "\x1b[38;2;255;0;0m"
	inBlue  = "\x1b[38;2;0;0;255m"
	inWhite = "\x1b[38;2;255;255;255m"
)

func separatorSegment(alias, background, leading, trailing string) *config.Segment {
	return &config.Segment{
		Type:          config.TEXT,
		Alias:         alias,
		Template:      alias,
		Background:    color.Ansi(background),
		Foreground:    "#ffffff",
		LeadingGlyph:  leading,
		TrailingGlyph: trailing,
	}
}

func renderTwoSegments(t *testing.T, first, second *config.Segment) string {
	t.Helper()

	engine := newSeparatorTestEngine(t, &config.LayoutConfig{
		Prompt:   []config.PromptLayout{{Segments: []string{"a", "b"}}},
		Segments: map[string]*config.Segment{"a": first, "b": second},
	})

	return engine.Primary()
}

// A segment that opens flat and closes with a separator is a link in the
// ribbon, so its neighbour's separator is drawn on its background and the two
// blocks meet with no gap.
func TestSeparatorJoinsTheRibbonIntoAShapedSegment(t *testing.T) {
	got := renderTwoSegments(t,
		separatorSegment("a", red, "", ">"),
		separatorSegment("b", blue, "", ">"),
	)

	require.Contains(t, got, onBlue+inRed+">", "a's separator should be drawn on b's background")
}

// A segment with no separators is bare text that happens to have colors, not a
// link in the ribbon. Its neighbour's separator keeps its own outline against
// the terminal instead of merging into a block of color.
//
// Several bundled themes rely on this: they draw their own opening inside the
// template with <transparent> markup, and a filled separator would put a block
// of color behind that outline.
func TestSeparatorFloatsBeforeASegmentWithNoSeparators(t *testing.T) {
	got := renderTwoSegments(t,
		separatorSegment("a", red, "", ">"),
		separatorSegment("b", blue, "", ""),
	)

	require.Contains(t, got, inRed+">", "a's separator should keep its outline")
	require.NotContains(t, got, onBlue+inRed+">", "a's separator must not take b's background")
}

// A segment that opens with its own separator is already drawing that edge, so
// filling the one before it would stack two shapes in the same place.
func TestSeparatorFloatsBeforeASegmentThatOpensWithItsOwn(t *testing.T) {
	got := renderTwoSegments(t,
		separatorSegment("a", red, "", ">"),
		separatorSegment("b", blue, "<", ">"),
	)

	require.Contains(t, got, inRed+">", "a's separator should keep its outline")
	require.NotContains(t, got, onBlue+inRed+">", "a's separator must not take b's background")
}

// A shaped segment that stops flat still shows its background at the boundary,
// so the next segment's opening separator is carved out of it.
func TestLeadingSeparatorIsCarvedFromAFlatEndingNeighbour(t *testing.T) {
	got := renderTwoSegments(t,
		separatorSegment("a", red, "<", ""),
		separatorSegment("b", blue, "<", ""),
	)

	require.Contains(t, got, onRed+inBlue+"<", "b's separator should be carved from a's background")
}

// The neighbour's background is only usable when it has one. An empty
// background reaches the writer as "the active segment's background", which
// would draw b's separator in b's own color on b's own color: invisible.
func TestLeadingSeparatorStaysVisibleAfterABackgroundlessSegment(t *testing.T) {
	got := renderTwoSegments(t,
		separatorSegment("a", "", "<", ""),
		separatorSegment("b", blue, "<", ""),
	)

	require.Contains(t, got, inBlue+"<", "b's separator should be visible against the terminal")
	require.NotContains(t, got, onBlue+inBlue+"<", "b's separator must not be drawn on its own background")
}

// A segment with no separators is bare text in both directions: the segment
// after it opens against the terminal, not against its background. Without the
// leading-separator half of the shaped test, a coloured run of plain segments
// would silently start carving shapes out of each other.
func TestLeadingSeparatorFloatsAfterASegmentWithNoSeparators(t *testing.T) {
	got := renderTwoSegments(t,
		separatorSegment("a", red, "", ""),
		separatorSegment("b", blue, "<", ""),
	)

	require.Contains(t, got, inBlue+"<", "b's separator should keep its outline")
	require.NotContains(t, got, onRed+inBlue+"<", "b's separator must not be carved from bare text")
}
