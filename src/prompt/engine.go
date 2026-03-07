package prompt

import (
	"strings"
	"sync"

	"github.com/po1o/prompto/src/cache"
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
	Padding               int
	currentLineLength     int
	rpromptLength         int
	cacheMu               sync.Mutex
	sharedProviderMu      sync.Mutex
	streamingMu           sync.Mutex
	stateMu               sync.Mutex
	Plain                 bool
	forceRender           bool
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

	switch e.activeSegment.ResolveStyle() {
	case config.Plain, config.Powerline:
		terminal.Write(color.Background, color.Foreground, e.activeSegment.Text())
	case config.Diamond:
		background := color.Transparent

		if e.previousActiveSegment != nil && e.previousActiveSegment.HasEmptyDiamondAtEnd() {
			background = e.previousActiveSegment.ResolveBackground()
		}

		terminal.Write(background, color.Background, e.activeSegment.LeadingDiamond)
		terminal.Write(color.Background, color.Foreground, e.activeSegment.Text())
	case config.Accordion:
		if e.activeSegment.Enabled {
			terminal.Write(color.Background, color.Foreground, e.activeSegment.Text())
		}
	}

	e.previousActiveSegment = e.activeSegment

	terminal.SetParentColors(e.previousActiveSegment.ResolveBackground(), e.previousActiveSegment.ResolveForeground())
}

func (e *Engine) writeSeparator(final bool) {
	if e.activeSegment == nil {
		return
	}

	isCurrentDiamond := e.activeSegment.ResolveStyle() == config.Diamond
	if final && isCurrentDiamond {
		terminal.Write(color.Transparent, color.Background, e.activeSegment.TrailingDiamond)
		return
	}

	isPreviousDiamond := e.previousActiveSegment != nil && e.previousActiveSegment.ResolveStyle() == config.Diamond
	if isPreviousDiamond {
		e.adjustTrailingDiamondColorOverrides()
	}

	if isPreviousDiamond && isCurrentDiamond && e.activeSegment.LeadingDiamond == "" {
		terminal.Write(color.Background, color.ParentBackground, e.previousActiveSegment.TrailingDiamond)
		return
	}

	if isPreviousDiamond && len(e.previousActiveSegment.TrailingDiamond) > 0 {
		terminal.Write(color.Transparent, color.ParentBackground, e.previousActiveSegment.TrailingDiamond)
	}

	isPowerline := e.activeSegment.IsPowerline()

	shouldOverridePowerlineLeadingSymbol := func() bool {
		if !isPowerline {
			return false
		}

		if isPowerline && e.activeSegment.LeadingPowerlineSymbol == "" {
			return false
		}

		if e.previousActiveSegment != nil && e.previousActiveSegment.IsPowerline() {
			return false
		}

		return true
	}

	if shouldOverridePowerlineLeadingSymbol() {
		terminal.Write(color.Transparent, color.Background, e.activeSegment.LeadingPowerlineSymbol)
		return
	}

	resolvePowerlineSymbol := func() string {
		if isPowerline {
			return e.activeSegment.PowerlineSymbol
		}

		if e.previousActiveSegment != nil && e.previousActiveSegment.IsPowerline() {
			return e.previousActiveSegment.PowerlineSymbol
		}

		return ""
	}

	symbol := resolvePowerlineSymbol()
	if symbol == "" {
		return
	}

	bgColor := color.Background
	if final || !isPowerline {
		bgColor = color.Transparent
	}

	if e.activeSegment.ResolveStyle() == config.Diamond && e.activeSegment.LeadingDiamond == "" {
		bgColor = color.Background
	}

	if e.activeSegment.InvertPowerline || (e.previousActiveSegment != nil && e.previousActiveSegment.InvertPowerline) {
		terminal.Write(e.getPowerlineColor(), bgColor, symbol)
		return
	}

	terminal.Write(bgColor, e.getPowerlineColor(), symbol)
}

func (e *Engine) getPowerlineColor() color.Ansi {
	if e.previousActiveSegment == nil {
		return color.Transparent
	}

	if e.previousActiveSegment.ResolveStyle() == config.Diamond && e.previousActiveSegment.TrailingDiamond == "" {
		return e.previousActiveSegment.ResolveBackground()
	}

	if e.activeSegment.ResolveStyle() == config.Diamond && e.activeSegment.LeadingDiamond == "" {
		return e.previousActiveSegment.ResolveBackground()
	}

	if !e.previousActiveSegment.IsPowerline() {
		return color.Transparent
	}

	return e.previousActiveSegment.ResolveBackground()
}

func (e *Engine) adjustTrailingDiamondColorOverrides() {
	// as we now already adjusted the activeSegment, we need to change the value
	// of background and foreground to parentBackground and parentForeground
	// this will still break when using parentBackground and parentForeground as keywords
	// in a trailing diamond, but let's fix that when it happens as it requires either a rewrite
	// of the logic for diamonds or storing grandparents as well like one happy family.
	if e.previousActiveSegment == nil || e.previousActiveSegment.TrailingDiamond == "" {
		return
	}

	trailingDiamond := e.previousActiveSegment.TrailingDiamond
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
		newOverride := override
		switch override { //nolint:exhaustive
		case color.Foreground:
			newOverride = color.ParentForeground
		case color.Background:
			newOverride = color.ParentBackground
		}

		if override == newOverride {
			return
		}

		newAnchor := strings.Replace(match[terminal.ANCHOR], string(override), string(newOverride), 1)
		e.previousActiveSegment.TrailingDiamond = strings.Replace(e.previousActiveSegment.TrailingDiamond, anchor, newAnchor, 1)
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

	reload, _ := cache.Get[bool](cache.Device, config.RELOAD)
	cfg := config.Get(flags.ConfigPath, reload)
	if cfg.Layout == nil {
		cfg.Layout = &config.LayoutConfig{}
	}
	layoutCfg := cfg.Layout

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
	case shell.XONSH:
		// In Xonsh, the behavior of wrapping at the end of a prompt line is inconsistent across different operating systems.
		// On Windows, it wraps before the last cell on the terminal screen, that is, the last cell is never available for a prompt line.
		if env.GOOS() == runtime.WINDOWS {
			eng.rectifyTerminalWidth(-1)
		}
	case shell.ELVISH:
		// In Elvish, the case is similar to that in Xonsh.
		// However, on Windows, we have to reduce the terminal width by 1 again to ensure that newlines are displayed correctly.
		diff := -1
		if env.GOOS() == runtime.WINDOWS {
			diff = -2
		}
		eng.rectifyTerminalWidth(diff)
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
