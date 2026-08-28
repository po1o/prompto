package prompt

import (
	"strings"
	"sync"

	"github.com/po1o/prompto/src/color"
	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/log"
	"github.com/po1o/prompto/src/regex"
	"github.com/po1o/prompto/src/runtime"
	"github.com/po1o/prompto/src/shell"
	"github.com/po1o/prompto/src/template"
	"github.com/po1o/prompto/src/terminal"
)

var cycle *color.Cycle = &color.Cycle{}

type Engine struct {
	deviceCache           DeviceCache
	Env                   runtime.Environment
	folderCache           map[string]segmentRenderCache
	activeSegment         *config.Segment
	sharedProviderFactory map[config.SegmentType]sharedProviderFactory
	updateCallback        func(string)
	// restoreLeadingGlyph undoes the block-level leading glyph substitution made
	// in writeSegment. It is held until the block ends rather than run on
	// return, because the next segment still reads the substituted value.
	restoreLeadingGlyph   func()
	Config                *config.Config
	sessionCache          map[string]segmentRenderCache
	sharedProviders       map[config.SegmentType]*onceProvider[sharedExecutionResult]
	LayoutConfig          *config.LayoutConfig
	previousActiveSegment *config.Segment
	pendingSegments       map[string]bool
	cachedValues          map[string]string
	segmentCacheKeys      map[string]string
	segmentStates         map[string]*segmentAsyncState
	Overflow              config.Overflow
	rprompt               string
	prompt                strings.Builder
	streamingBlocks       []*config.Block
	streamingRTransient   []*config.Block
	streamingTransient    []*config.Block
	Padding               int
	currentLineLength     int
	rpromptLength         int
	cacheMu               sync.Mutex
	sharedProviderMu      sync.Mutex
	streamingMu           sync.Mutex
	stateMu               sync.Mutex
	executionWG           sync.WaitGroup
	Plain                 bool
	forceRender           bool
	repaintOnly           bool
	// vimRepainted, guarded by streamingMu, records that PrimaryRepaint has
	// re-executed the canonical vim segment since the current render
	// generation started; late async merges of vim results are then dropped.
	vimRepainted bool
}

const (
	PRIMARY   = "primary"
	TRANSIENT = "transient"
	DEBUG     = "debug"
	SECONDARY = "secondary"
	RIGHT     = "right"
	TOOLTIP   = "tooltip"
	VALID     = "valid"
	ERROR     = "error"
	PREVIEW   = "preview"
)

func (e *Engine) write(txt string) {
	// Grow capacity proactively if needed
	if e.prompt.Cap() < e.prompt.Len()+len(txt) {
		e.prompt.Grow(len(txt) * 2) // Grow by double the needed size to reduce future allocations
	}
	e.prompt.WriteString(txt)
}

func (e *Engine) string() string {
	txt := e.prompt.String()
	e.prompt.Reset()
	return txt
}

func (e *Engine) canWriteRightBlock(length int, rprompt bool) (int, bool) {
	if rprompt && (e.rprompt == "") {
		return 0, false
	}

	consoleWidth, err := e.Env.TerminalWidth()
	if err != nil || consoleWidth == 0 {
		return 0, false
	}

	availableSpace := consoleWidth - e.currentLineLength

	// spanning multiple lines
	if availableSpace < 0 {
		overflow := e.currentLineLength % consoleWidth
		availableSpace = consoleWidth - overflow
	}

	availableSpace -= length

	promptBreathingRoom := 5
	if rprompt {
		promptBreathingRoom = 30
	}

	canWrite := availableSpace >= promptBreathingRoom

	// reset the available space when we can't write so we can fill the line
	if !canWrite {
		availableSpace = consoleWidth - length
	}

	return availableSpace, canWrite
}

func (e *Engine) pwd() {
	// only print when relevant
	if e.Config.PWD == "" {
		return
	}

	// only print when supported
	sh := e.Env.Shell()
	if sh == shell.ELVISH || sh == shell.XONSH {
		return
	}

	pwd := e.Env.Pwd()
	if e.Env.IsCygwin() {
		pwd = strings.ReplaceAll(pwd, `\`, `/`)
	}

	// Allow template logic to define when to enable the PWD (when supported)
	pwdType, err := template.Render(e.Config.PWD, nil)
	if err != nil || pwdType == "" {
		return
	}

	// Convert to Windows path when in WSL
	if e.Env.IsWsl() {
		pwd = e.Env.ConvertToWindowsPath(pwd)
	}

	user := e.Env.User()
	host, _ := e.Env.Host()
	e.write(terminal.Pwd(pwdType, user, host, pwd))
}

func (e *Engine) getNewline() string {
	newline := "\n"

	if e.Plain || e.Env.Flags().Debug {
		return newline
	}

	// Warp terminal will remove a newline character ('\n') from the prompt, so we hack it in.
	if e.isWarp() {
		return terminal.LineBreak()
	}

	return newline
}

func (e *Engine) writeNewline() {
	defer func() {
		e.currentLineLength = 0
	}()

	e.write(e.getNewline())
}

func (e *Engine) isWarp() bool {
	return terminal.Program == terminal.Warp
}

func (e *Engine) isIterm() bool {
	return terminal.Program == terminal.ITerm
}

func (e *Engine) shouldFill(filler string, padLength int) (string, bool) {
	if filler == "" {
		log.Debug("no filler specified")
		return "", false
	}

	e.Padding = padLength

	defer func() {
		e.Padding = 0
	}()

	var err error
	if filler, err = template.Render(filler, e); err != nil {
		return "", false
	}

	// allow for easy color overrides and templates
	terminal.SetColors("default", "default")
	terminal.Write("", "", filler)
	filler, lenFiller := terminal.String()
	if lenFiller == 0 {
		log.Debug("filler has no length")
		return "", false
	}

	repeat := padLength / lenFiller
	unfilled := padLength % lenFiller
	txt := strings.Repeat(filler, repeat) + strings.Repeat(" ", unfilled)
	log.Debug("filling with", txt)
	return txt, true
}

func (e *Engine) getTitleTemplateText() string {
	if txt, err := template.Render(e.Config.ConsoleTitleTemplate, nil); err == nil {
		return txt
	}

	return ""
}

func (e *Engine) renderBlock(block *config.Block, cancelNewline bool) bool {
	blockText, length := e.writeBlockSegments(block)
	return e.renderBlockWithText(block, blockText, length, cancelNewline)
}

func (e *Engine) appendRightPromptLine(blockText string, length int, newline bool) bool {
	if !newline && blockText == "" {
		e.rprompt = ""
		e.rpromptLength = 0
		return false
	}

	if newline {
		e.rprompt += "\n"
	}

	e.rprompt += blockText
	e.rpromptLength = max(e.rpromptLength, length)
	return true
}

func (e *Engine) applyPowerShellBleedPatch() {
	// when in PowerShell, we need to clear the line after the prompt
	// to avoid the background being printed on the next line
	// when at the end of the buffer.
	// See https://github.com/po1o/prompto/issues/65
	if e.Env.Shell() != shell.PWSH {
		return
	}

	// only do this when enabled
	if !e.Config.PatchPwshBleed {
		return
	}

	e.write(terminal.ClearAfter())
}

func (e *Engine) setActiveSegment(segment *config.Segment) {
	e.activeSegment = segment
	terminal.Interactive = segment.Interactive
	terminal.SetColors(segment.ResolveBackground(), segment.ResolveForeground())
}

func (e *Engine) renderActiveSegment() {
	e.writeSeparator(false)

	background := color.Transparent

	// A previous segment that ends flat still shows its background at the
	// boundary, so the leading glyph is carved out of it. An empty background
	// must stay empty here: the writer reads that as "the active segment's
	// background", which would paint the glyph onto itself and hide it.
	if e.previousActiveSegment != nil && e.previousActiveSegment.HasEmptyGlyphAtEnd() {
		if previous := e.previousActiveSegment.ResolveBackground(); previous != "" {
			background = previous
		}
	}

	terminal.Write(background, e.separatorForeground(e.activeSegment, color.Background), e.activeSegment.LeadingGlyph)

	// A segment kept via keep_when_empty draws its glyphs but no text, so the
	// prompt holds its shape instead of reflowing.
	if e.activeSegment.Enabled {
		terminal.Write(color.Background, color.Foreground, e.activeSegment.Text())
	}

	e.previousActiveSegment = e.activeSegment

	terminal.SetParentColors(e.previousActiveSegment.ResolveBackground(), e.previousActiveSegment.ResolveForeground())
}

// separatorForeground is the color the given segment's separator glyphs take.
//
// fallback is the keyword this call site used before separator_foreground
// existed, and it differs per site: a segment's own glyphs resolve against its
// background, while the trailing glyph of the segment before it resolves
// against the parent's. Passing it through keeps every graphical theme byte
// identical, since the keyword is still what reaches the writer.
//
// The console fallback is a property of the config rather than of TERM: the
// daemon serves console and desktop sessions at once and cannot tell them
// apart, so each session's own config carries the answer.
func (e *Engine) separatorForeground(segment *config.Segment, fallback color.Ansi) color.Ansi {
	if segment.SeparatorForeground != "" {
		return segment.SeparatorForeground
	}

	if e.LayoutConfig == nil || !e.LayoutConfig.Console {
		return fallback
	}

	return segment.ResolveSeparatorForeground(true)
}

func (e *Engine) writeSeparator(final bool) {
	if e.activeSegment == nil {
		return
	}

	if final {
		terminal.Write(color.Transparent, e.separatorForeground(e.activeSegment, color.Background), e.activeSegment.TrailingGlyph)
		return
	}

	if e.previousActiveSegment == nil || e.previousActiveSegment.TrailingGlyph == "" {
		return
	}

	e.adjustTrailingGlyphColorOverrides()

	// A shaped segment that opens flat lets the previous trailing glyph land on
	// its own background, which is what makes a run of segments read as one
	// continuous ribbon. A segment carrying no glyph is bare text rather than a
	// link in that ribbon, so the glyph closes onto the terminal background and
	// keeps its shape instead of merging into a block of colour.
	if e.activeSegment.LeadingGlyph == "" && e.activeSegment.TrailingGlyph != "" {
		terminal.Write(color.Background, e.separatorForeground(e.previousActiveSegment, color.ParentBackground), e.previousActiveSegment.TrailingGlyph)
		return
	}

	terminal.Write(color.Transparent, e.separatorForeground(e.previousActiveSegment, color.ParentBackground), e.previousActiveSegment.TrailingGlyph)
}

func (e *Engine) adjustTrailingGlyphColorOverrides() {
	// as we now already adjusted the activeSegment, we need to change the value
	// of background and foreground to parentBackground and parentForeground
	// this will still break when using parentBackground and parentForeground as keywords
	// in a trailing diamond, but let's fix that when it happens as it requires either a rewrite
	// of the logic for diamonds or storing grandparents as well like one happy family.
	//
	// The caller has already established that previousActiveSegment carries a
	// trailing glyph, so this reads it directly.
	trailingDiamond := e.previousActiveSegment.TrailingGlyph
	// Optimize: check both conditions in a single pass
	hasBg := strings.Contains(trailingDiamond, string(color.Background))
	hasFg := strings.Contains(trailingDiamond, string(color.Foreground))

	if !hasBg && !hasFg {
		return
	}

	match := regex.FindNamedRegexMatch(terminal.AnchorRegex, trailingDiamond)
	if len(match) == 0 {
		return
	}

	adjustOverride := func(anchor string, override color.Ansi) {
		if override != color.Foreground && override != color.Background {
			return
		}

		newOverride := color.ParentForeground
		if override == color.Background {
			newOverride = color.ParentBackground
		}

		newAnchor := strings.Replace(match[terminal.ANCHOR], string(override), string(newOverride), 1)
		e.previousActiveSegment.TrailingGlyph = strings.Replace(e.previousActiveSegment.TrailingGlyph, anchor, newAnchor, 1)
	}

	if len(match[terminal.BG]) > 0 {
		adjustOverride(match[terminal.ANCHOR], color.Ansi(match[terminal.BG]))
	}

	if len(match[terminal.FG]) > 0 {
		adjustOverride(match[terminal.ANCHOR], color.Ansi(match[terminal.FG]))
	}
}

func (e *Engine) rectifyTerminalWidth(diff int) {
	// Since the terminal width may not be given by the CLI flag, we should always call this here.
	_, err := e.Env.TerminalWidth()
	if err != nil {
		// Skip when we're unable to determine the terminal width.
		return
	}

	e.Env.Flags().TerminalWidth += diff
}

// New returns a prompt engine initialized with the
// given configuration options, and is ready to print any
// of the prompt components.
func New(flags *runtime.Flags) *Engine {
	env := &runtime.Terminal{}
	env.Init(flags)

	cfg := config.Load(flags.ConfigPath)
	if cfg.Layout == nil {
		cfg.Layout = &config.LayoutConfig{}
	}
	layoutCfg := cfg.Layout
	flags.IsPrimary = flags.Type == "" || flags.Type == PRIMARY

	template.Init(env, cfg.Var, cfg.Maps)

	flags.HasExtra = cfg.HasSecondary || cfg.HasTransient || cfg.ValidLine != nil || cfg.ErrorLine != nil || cfg.DebugPrompt != nil

	// when we print using https://github.com/akinomyoga/ble.sh, this needs to be unescaped for certain prompts
	sh := env.Shell()
	if sh == shell.BASH && !flags.Escape {
		sh = shell.GENERIC
	}

	terminal.Init(sh)
	terminal.BackgroundColor = cfg.TerminalBackground.ResolveTemplate()
	terminal.Colors = cfg.MakeColors(env)
	terminal.Plain = flags.Plain

	eng := &Engine{
		Config:                cfg,
		Env:                   env,
		Plain:                 flags.Plain,
		forceRender:           flags.Force || len(env.Getenv("PROMPTO_FORCE_RENDER")) > 0,
		LayoutConfig:          layoutCfg,
		sharedProviderFactory: defaultSharedProviderFactories(),
		segmentStates:         make(map[string]*segmentAsyncState),
		sessionCache:          make(map[string]segmentRenderCache),
		folderCache:           make(map[string]segmentRenderCache),
		prompt:                strings.Builder{},
	}

	// Pre-allocate prompt builder capacity to reduce allocations during rendering
	eng.prompt.Grow(512) // Start with 512 bytes capacity, will grow as needed

	switch env.Shell() {
	case shell.ELVISH:
		// In Elvish, the behavior of wrapping at the end of a prompt line is inconsistent.
		eng.rectifyTerminalWidth(-1)
	case shell.PWSH:
		// when in PowerShell, and force patching the bleed bug
		// we need to reduce the terminal width by 1 so the last
		// character isn't cut off by the ANSI escape sequences
		// See https://github.com/po1o/prompto/issues/65
		if cfg.PatchPwshBleed {
			eng.rectifyTerminalWidth(-1)
		}
	}

	return eng
}
