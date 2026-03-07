package config

import (
	"encoding/gob"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/cli/upgrade"
	"github.com/po1o/prompto/src/color"
	"github.com/po1o/prompto/src/log"
	"github.com/po1o/prompto/src/maps"
	"github.com/po1o/prompto/src/runtime"
	"github.com/po1o/prompto/src/segments"
	"github.com/po1o/prompto/src/shell"
	"github.com/po1o/prompto/src/template"
	"github.com/po1o/prompto/src/terminal"
)

func init() {
	gob.Register(&Config{})
}

const (
	YAML string = "yaml"
	YML  string = "yml"

	AUTOUPGRADE   = "upgrade"
	UPGRADENOTICE = "notice"
	RELOAD        = "reload"
)

type Action string

func (a Action) IsDefault() bool {
	return a != Prepend && a != Extend
}

const (
	Prepend Action = "prepend"
	Extend  Action = "extend"
)

// VimConfig holds vim mode settings.
type VimConfig struct {
	Enabled     bool `yaml:"enabled,omitempty"`
	CursorShape bool `yaml:"cursor_shape,omitempty"`
	CursorBlink bool `yaml:"cursor_blink,omitempty"`
}

// Config holds all the theme for rendering the prompt
type Config struct {
	Palette                 color.Palette          `yaml:"palette,omitempty"`
	DebugPrompt             *Segment               `yaml:"debug_prompt,omitempty"`
	Var                     map[string]any         `yaml:"var,omitempty"`
	Palettes                *color.Palettes        `yaml:"palettes,omitempty"`
	ValidLine               *Segment               `yaml:"valid_line,omitempty"`
	ErrorLine               *Segment               `yaml:"error_line,omitempty"`
	Maps                    *maps.Config           `yaml:"maps,omitempty"`
	Upgrade                 *upgrade.Config        `yaml:"upgrade,omitempty"`
	VimMode                 *VimConfig             `yaml:"vim-mode,omitempty"`
	Layout                  *LayoutConfig          `yaml:"-"`
	Source                  string                 `yaml:"-"`
	DaemonIdleTimeout       string                 `yaml:"daemon_idle_timeout,omitempty"`
	RenderPendingIcon       string                 `yaml:"render_pending_icon,omitempty"`
	RenderPendingBackground color.Ansi             `yaml:"render_pending_background,omitempty"`
	ConsoleTitleTemplate    string                 `yaml:"console_title_template,omitempty"`
	PWD                     string                 `yaml:"pwd,omitempty"`
	AccentColor             color.Ansi             `yaml:"accent_color,omitempty"`
	Format                  string                 `yaml:"-"`
	TerminalBackground      color.Ansi             `yaml:"terminal_background,omitempty"`
	ToolTipsAction          Action                 `yaml:"tooltips_action,omitempty"`
	FilePaths               []string               `yaml:"-"`
	Tooltips                []*Segment             `yaml:"tooltips,omitempty"`
	Cycle                   color.Cycle            `yaml:"cycle,omitempty"`
	ITermFeatures           terminal.ITermFeatures `yaml:"iterm_features,omitempty"`
	DaemonTimeout           int                    `yaml:"daemon_timeout,omitempty"`
	hash                    uint64
	Async                   bool `yaml:"async,omitempty"`
	HasTransient            bool `yaml:"-"`
	ShellIntegration        bool `yaml:"shell_integration,omitempty"`
	FinalSpace              bool `yaml:"final_space,omitempty"`
	UpgradeNotice           bool `yaml:"-"`
	PatchPwshBleed          bool `yaml:"patch_pwsh_bleed,omitempty"`
	AutoUpgrade             bool `yaml:"-"`
	EnableCursorPositioning bool `yaml:"enable_cursor_positioning,omitempty"`
	HasSecondary            bool `yaml:"-"`
}

func (cfg *Config) MakeColors(env runtime.Environment) color.String {
	cacheDisabled := env.Getenv("PROMPTO_CACHE_DISABLED") == "1"
	return color.MakeColors(cfg.getPalette(), !cacheDisabled, cfg.AccentColor, env)
}

func (cfg *Config) getPalette() color.Palette {
	if cfg.Palettes == nil {
		return cfg.Palette
	}

	key, err := template.Render(cfg.Palettes.Template, nil)
	if err != nil {
		return cfg.Palette
	}

	palette, ok := cfg.Palettes.List[key]
	if !ok {
		return cfg.Palette
	}

	for key, color := range cfg.Palette {
		if _, ok := palette[key]; ok {
			continue
		}

		palette[key] = color
	}

	return palette
}

func (cfg *Config) Features(env runtime.Environment, daemon bool) shell.Features {
	var feats shell.Features

	asyncShells := []string{shell.BASH, shell.ZSH, shell.FISH, shell.PWSH}

	if cfg.Async && slices.Contains(asyncShells, env.Shell()) {
		log.Debug("async enabled")
		feats |= shell.Async
	}

	if daemon && slices.Contains(asyncShells, env.Shell()) {
		log.Debug("daemon enabled")
		feats |= shell.Daemon
	}

	if cfg.HasTransient {
		log.Debug("transient prompt enabled")
		feats |= shell.Transient
	}

	unsupportedShells := []string{shell.ELVISH, shell.XONSH}
	if slices.Contains(unsupportedShells, env.Shell()) {
		cfg.ShellIntegration = false
	}

	if cfg.ShellIntegration {
		log.Debug("shell integration enabled")
		feats |= shell.FTCSMarks
	}

	// do not enable upgrade features when async is enabled
	if feats&shell.Async == 0 {
		feats |= cfg.upgradeFeatures()
	}

	if cfg.ErrorLine != nil || cfg.ValidLine != nil {
		log.Debug("error or valid line enabled")
		feats |= shell.LineError
	}

	if len(cfg.Tooltips) > 0 {
		log.Debug("tooltips enabled")
		feats |= shell.Tooltips
	}

	if env.Shell() == shell.FISH && cfg.ITermFeatures != nil && cfg.ITermFeatures.Contains(terminal.PromptMark) {
		log.Debug("prompt mark enabled")
		feats |= shell.PromptMark
	}

	if cfg.EnableCursorPositioning && cfg.hasNewlineSegmentInFirstPromptLine() {
		log.Debug("cursor positioning enabled")
		feats |= shell.CursorPositioning
	}

	if cfg.Layout != nil && len(cfg.Layout.RPrompt) > 0 {
		feats |= shell.RPrompt
	}

	if cfg.Layout != nil {
		for _, segment := range cfg.Layout.Segments {
			if segment.Type == AZ {
				source := segment.Options.String(segments.Source, segments.FirstMatch)
				if strings.Contains(source, segments.Pwsh) {
					log.Debug("azure enabled")
					feats |= shell.Azure
				}
			}

			if segment.Type == GIT {
				source := segment.Options.String(segments.Source, segments.Cli)
				if source == segments.Pwsh {
					log.Debug("posh-git enabled")
					feats |= shell.PoshGit
				}
			}
		}
	}

	if cfg.VimMode != nil {
		feats |= cfg.vimFeatures(cfg.VimMode)
	}

	return feats
}

func (cfg *Config) hasNewlineSegmentInFirstPromptLine() bool {
	if cfg == nil || cfg.Layout == nil || len(cfg.Layout.Prompt) == 0 {
		return false
	}

	first := cfg.Layout.Prompt[0]
	for _, name := range first.Segments {
		segment, ok := cfg.Layout.Segments[name]
		if !ok {
			continue
		}

		if segment.Newline {
			return true
		}
	}

	return false
}

func (cfg *Config) upgradeFeatures() shell.Features {
	var feats shell.Features

	autoUpgrade := cfg.Upgrade.Auto
	if val, OK := cache.Get[bool](cache.Device, AUTOUPGRADE); OK {
		log.Debug("auto upgrade key found, overriding config")
		autoUpgrade = val
	}

	upgradeNotice := cfg.Upgrade.DisplayNotice
	if val, OK := cache.Get[bool](cache.Device, UPGRADENOTICE); OK {
		log.Debug("upgrade notice key found, overriding config")
		upgradeNotice = val
	}

	if upgradeNotice && !autoUpgrade {
		log.Debug("notice enabled, no auto upgrade")
		feats |= shell.Notice
	}

	if autoUpgrade {
		log.Debug("auto upgrade enabled")
		feats |= shell.Upgrade
	}

	return feats
}

func (cfg *Config) vimFeatures(vimCfg *VimConfig) shell.Features {
	var feats shell.Features

	cursorControl := vimCfg.CursorShape || vimCfg.CursorBlink

	if vimCfg.Enabled || cursorControl {
		log.Debug("vim mode enabled")
		feats |= shell.VimMode
	}

	if cursorControl {
		log.Debug("vim cursor shape enabled")
		feats |= shell.VimCursorShape
	}

	if vimCfg.CursorBlink {
		log.Debug("vim cursor blink enabled")
		feats |= shell.VimCursorBlink
	}

	return feats
}

func (cfg *Config) Hash() uint64 {
	return cfg.hash
}

// GetDaemonIdleTimeout returns the daemon idle timeout duration.
// Returns 0 when daemon idle shutdown should be disabled.
// Defaults to 5 minutes when unset or invalid.
func (cfg *Config) GetDaemonIdleTimeout() time.Duration {
	if cfg.DaemonIdleTimeout == "" {
		return 5 * time.Minute
	}

	if cfg.DaemonIdleTimeout == "none" {
		return 0
	}

	minutes, err := strconv.Atoi(cfg.DaemonIdleTimeout)
	if err != nil || minutes < 0 {
		log.Debugf("invalid daemon_idle_timeout value %q, defaulting to 5 minutes", cfg.DaemonIdleTimeout)
		return 5 * time.Minute
	}

	return time.Duration(minutes) * time.Minute
}

// GetDaemonTimeout returns the timeout for switching from initial to streamed daemon updates.
// Defaults to 100 milliseconds when unset or invalid.
func (cfg *Config) GetDaemonTimeout() time.Duration {
	if cfg == nil || cfg.DaemonTimeout <= 0 {
		return 100 * time.Millisecond
	}

	return time.Duration(cfg.DaemonTimeout) * time.Millisecond
}

// toggleSegments processes all layout segments and adds segments
// with Toggled == true to the toggle cache, effectively toggling them off.
func (cfg *Config) toggleSegments() {
	currentToggleSet, _ := cache.Get[map[string]bool](cache.Session, cache.TOGGLECACHE)
	if currentToggleSet == nil {
		currentToggleSet = make(map[string]bool)
	}

	if cfg.Layout != nil {
		for _, segment := range cfg.Layout.Segments {
			if !segment.Toggled {
				continue
			}

			segmentName := segment.Alias
			if segmentName == "" {
				segmentName = string(segment.Type)
			}

			currentToggleSet[segmentName] = true
		}
	}

	// Update cache with the map directly
	cache.Set(cache.Session, cache.TOGGLECACHE, currentToggleSet, cache.INFINITE)
}
