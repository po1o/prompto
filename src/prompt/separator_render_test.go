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

// A line's own leading separator stands in for the first segment's, so that
// segment is shaped even though its definition carries no leading glyph. The
// segment after it must see that: it carves its opening out of a flat-ending
// shaped neighbour, and a segment the line opened for is exactly that.
//
// The restore of the substituted glyph therefore cannot happen when the first
// segment is done rendering — the second one has not read it yet.
func TestLeadingSeparatorIsCarvedFromANeighbourTheLineOpenedFor(t *testing.T) {
	first := separatorSegment("a", red, "", "")
	second := separatorSegment("b", blue, "<", "")

	engine := newSeparatorTestEngine(t, &config.LayoutConfig{
		Prompt:   []config.PromptLayout{{Segments: []string{"a", "b"}, LeadingGlyph: "<"}},
		Segments: map[string]*config.Segment{"a": first, "b": second},
	})

	got := engine.Primary()

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

// A separator is the segment's own block shaped into a point, so it is drawn in
// the segment's background color. Without one there is nothing to draw it with:
// the trailing separator is written on the terminal in the terminal's own
// color and disappears. Documented in docs/configuration/segments.md under
// "Separators Need a Background", and pinned here because nothing warns about
// it — a trailing_separator simply never appears.
func TestSeparatorsNeedABackgroundToBeDrawn(t *testing.T) {
	render := func(background string) string {
		return renderTwoSegments(t,
			separatorSegment("a", background, "(", ")"),
			separatorSegment("b", blue, "", ""),
		)
	}

	withBackground := render(red)
	require.Contains(t, withBackground, "(", "a leading separator should be drawn")
	require.Contains(t, withBackground, ")", "a trailing separator should be drawn")

	// No background: the leading separator falls back to the terminal's default
	// foreground, the trailing one is lost.
	none := render("")
	require.Contains(t, none, "(", "the leading separator survives on the default foreground")
	require.NotContains(t, none, ")", "the trailing separator has no color to be drawn in")

	// Explicitly transparent: neither is drawn.
	transparent := render("transparent")
	require.NotContains(t, transparent, "(")
	require.NotContains(t, transparent, ")")
}

// A console theme leaves backgrounds transparent, and a separator drawn in a
// transparent background is drawn in nothing at all. On a console config it
// falls back to the segment's own foreground so the shape survives.
func TestConsoleSeparatorFallsBackToTheForeground(t *testing.T) {
	segment := separatorSegment("a", "", "", ">")
	segment.Background = "transparent"

	engine := newSeparatorTestEngine(t, &config.LayoutConfig{
		Console:  true,
		Prompt:   []config.PromptLayout{{Segments: []string{"a"}}},
		Segments: map[string]*config.Segment{"a": segment},
	})

	require.Contains(t, engine.Primary(), inWhite+">", "the separator should take the segment's foreground")
}

// The same config without the console flag keeps the old rule, where the
// separator takes the background and a transparent one draws nothing. Eleven
// bundled themes depend on that.
func TestSeparatorStaysHiddenOnATransparentBackgroundOffConsole(t *testing.T) {
	segment := separatorSegment("a", "", "", ">")
	segment.Background = "transparent"

	engine := newSeparatorTestEngine(t, &config.LayoutConfig{
		Prompt:   []config.PromptLayout{{Segments: []string{"a"}}},
		Segments: map[string]*config.Segment{"a": segment},
	})

	require.NotContains(t, engine.Primary(), ">", "a transparent background draws no separator")
}

// separator_foreground is the explicit override, and it wins over both rules —
// including on a graphical theme, where it is the only way to color a separator
// differently from the block it closes.
func TestSeparatorForegroundOverridesTheBackground(t *testing.T) {
	segment := separatorSegment("a", red, "", ">")
	segment.SeparatorForeground = "#0000ff"

	engine := newSeparatorTestEngine(t, &config.LayoutConfig{
		Prompt:   []config.PromptLayout{{Segments: []string{"a"}}},
		Segments: map[string]*config.Segment{"a": segment},
	})

	got := engine.Primary()
	require.Contains(t, got, inBlue+">", "the separator should take separator_foreground")
	require.NotContains(t, got, inRed+">", "not the segment's background")
}
