package config

import (
	"fmt"
	"maps"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/po1o/prompto/src/cli/upgrade"
	"github.com/po1o/prompto/src/color"
	configmaps "github.com/po1o/prompto/src/maps"
	"github.com/po1o/prompto/src/terminal"
	yaml "go.yaml.in/yaml/v3"
)

type PromptLayout struct {
	Style             string   `yaml:"style,omitempty"`
	Filler            string   `yaml:"filler,omitempty"`
	LeadingStyle      string   `yaml:"leading_style,omitempty"`
	TrailingStyle     string   `yaml:"trailing_style,omitempty"`
	LeadingSeparator  string   `yaml:"leading_separator,omitempty"`
	TrailingSeparator string   `yaml:"trailing_separator,omitempty"`
	LeadingDiamond    string   `yaml:"leading_diamond,omitempty"`
	TrailingDiamond   string   `yaml:"trailing_diamond,omitempty"`
	Segments          []string `yaml:"segments,omitempty"`
}

type LayoutConfig struct {
	Palette                 color.Palette          `yaml:"palette,omitempty"`
	Var                     map[string]any         `yaml:"var,omitempty"`
	Palettes                *color.Palettes        `yaml:"palettes,omitempty"`
	Maps                    *configmaps.Config     `yaml:"maps,omitempty"`
	Upgrade                 *upgrade.Config        `yaml:"upgrade,omitempty"`
	Cycle                   color.Cycle            `yaml:"cycle,omitempty"`
	ITermFeatures           terminal.ITermFeatures `yaml:"iterm_features,omitempty"`
	VimMode                 *VimConfig             `yaml:"vim-mode,omitempty"`
	AccentColor             color.Ansi             `yaml:"accent_color,omitempty"`
	DaemonIdleTimeout       string                 `yaml:"daemon_idle_timeout,omitempty"`
	RenderPendingIcon       string                 `yaml:"render_pending_icon,omitempty"`
	RenderPendingBackground color.Ansi             `yaml:"render_pending_background,omitempty"`
	ConsoleTitleTemplate    string                 `yaml:"console_title_template,omitempty"`
	PWD                     string                 `yaml:"pwd,omitempty"`
	TerminalBackground      color.Ansi             `yaml:"terminal_background,omitempty"`
	ToolTipsAction          Action                 `yaml:"tooltips_action,omitempty"`
	Tooltips                []*Segment             `yaml:"tooltips,omitempty"`
	DebugPrompt             *Segment               `yaml:"debug_prompt,omitempty"`
	ValidLine               *Segment               `yaml:"valid_line,omitempty"`
	ErrorLine               *Segment               `yaml:"error_line,omitempty"`
	Segments                map[string]*Segment    `yaml:"-"`
	Source                  string                 `yaml:"-"`
	Prompt                  []PromptLayout         `yaml:"prompt,omitempty"`
	RPrompt                 []PromptLayout         `yaml:"rprompt,omitempty"`
	SecondaryPrompt         []PromptLayout         `yaml:"secondary,omitempty"`
	TransientPrompt         []PromptLayout         `yaml:"transient,omitempty"`
	TransientRPrompt        []PromptLayout         `yaml:"rtransient,omitempty"`
	DaemonTimeout           int                    `yaml:"daemon_timeout,omitempty"`
	Async                   bool                   `yaml:"async,omitempty"`
	ShellIntegration        bool                   `yaml:"shell_integration,omitempty"`
	CursorPadding           bool                   `yaml:"cursor_padding,omitempty"`
	PatchPwshBleed          bool                   `yaml:"patch_pwsh_bleed,omitempty"`
	EnableCursorPositioning bool                   `yaml:"enable_cursor_positioning,omitempty"`
}

type layoutRawConfig struct {
	Palette                 color.Palette          `yaml:"palette"`
	Var                     map[string]any         `yaml:"var"`
	Palettes                *color.Palettes        `yaml:"palettes"`
	Maps                    *configmaps.Config     `yaml:"maps"`
	Upgrade                 *upgrade.Config        `yaml:"upgrade"`
	CursorPadding           *bool                  `yaml:"cursor_padding"`
	VimMode                 *VimConfig             `yaml:"vim-mode"`
	ErrorLine               *Segment               `yaml:"error_line"`
	ValidLine               *Segment               `yaml:"valid_line"`
	DebugPrompt             *Segment               `yaml:"debug_prompt"`
	AccentColor             color.Ansi             `yaml:"accent_color"`
	ConsoleTitleTemplate    string                 `yaml:"console_title_template"`
	PWD                     string                 `yaml:"pwd"`
	TerminalBackground      color.Ansi             `yaml:"terminal_background"`
	ToolTipsAction          Action                 `yaml:"tooltips_action"`
	RenderPendingBackground color.Ansi             `yaml:"render_pending_background"`
	RenderPendingIcon       string                 `yaml:"render_pending_icon"`
	DaemonIdleTimeout       string                 `yaml:"daemon_idle_timeout"`
	Tooltips                []*Segment             `yaml:"tooltips"`
	Prompt                  []PromptLayout         `yaml:"prompt"`
	RPrompt                 []PromptLayout         `yaml:"rprompt"`
	Secondary               []PromptLayout         `yaml:"secondary"`
	Transient               []PromptLayout         `yaml:"transient"`
	RTransient              []PromptLayout         `yaml:"rtransient"`
	ITermFeatures           terminal.ITermFeatures `yaml:"iterm_features"`
	Cycle                   color.Cycle            `yaml:"cycle"`
	DaemonTimeout           int                    `yaml:"daemon_timeout"`
	Async                   bool                   `yaml:"async"`
	ShellIntegration        bool                   `yaml:"shell_integration"`
	PatchPwshBleed          bool                   `yaml:"patch_pwsh_bleed"`
	EnableCursorPositioning bool                   `yaml:"enable_cursor_positioning"`
}

var knownLayoutTopLevelKeys = func() map[string]bool {
	keys := make(map[string]bool)
	rawType := reflect.TypeFor[layoutRawConfig]()

	for field := range rawType.Fields() {
		tag := field.Tag.Get("yaml")
		key, _, _ := strings.Cut(tag, ",")
		if key == "" || key == "-" {
			continue
		}

		keys[key] = true
	}

	return keys
}()

func LoadLayout(configFile string) (*LayoutConfig, error) {
	if configFile == "" {
		return nil, ErrNoConfig
	}

	configFile = resolveConfigLocation(configFile)
	format := strings.TrimPrefix(filepath.Ext(configFile), ".")
	if format != YAML && format != YML {
		return nil, ErrInvalidExtension
	}

	data, err := getData(configFile)
	if err != nil {
		return nil, ErrFileNotFound
	}

	cfg, err := ParseLayoutYAML(data)
	if err != nil {
		return nil, err
	}

	cfg.Source = configFile

	return cfg, nil
}

func ParseLayoutYAML(data []byte) (*LayoutConfig, error) {
	var raw layoutRawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, ErrParse
	}

	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, ErrParse
	}

	if err := validateLayoutTopLevelKeys(doc); err != nil {
		return nil, err
	}

	cursorPadding := resolveCursorPadding(&raw)

	layout := &LayoutConfig{
		Palette:                 raw.Palette,
		Var:                     raw.Var,
		Palettes:                raw.Palettes,
		Maps:                    raw.Maps,
		Upgrade:                 raw.Upgrade,
		Cycle:                   raw.Cycle,
		ITermFeatures:           raw.ITermFeatures,
		VimMode:                 raw.VimMode,
		AccentColor:             raw.AccentColor,
		DaemonIdleTimeout:       raw.DaemonIdleTimeout,
		RenderPendingIcon:       raw.RenderPendingIcon,
		RenderPendingBackground: raw.RenderPendingBackground,
		ConsoleTitleTemplate:    raw.ConsoleTitleTemplate,
		PWD:                     raw.PWD,
		TerminalBackground:      raw.TerminalBackground,
		ToolTipsAction:          raw.ToolTipsAction,
		Tooltips:                raw.Tooltips,
		DebugPrompt:             raw.DebugPrompt,
		ValidLine:               raw.ValidLine,
		ErrorLine:               raw.ErrorLine,
		Prompt:                  raw.Prompt,
		RPrompt:                 raw.RPrompt,
		SecondaryPrompt:         raw.Secondary,
		TransientPrompt:         raw.Transient,
		TransientRPrompt:        raw.RTransient,
		DaemonTimeout:           raw.DaemonTimeout,
		Async:                   raw.Async,
		ShellIntegration:        raw.ShellIntegration,
		CursorPadding:           cursorPadding,
		PatchPwshBleed:          raw.PatchPwshBleed,
		EnableCursorPositioning: raw.EnableCursorPositioning,
		Segments:                make(map[string]*Segment),
	}

	if err := normalizePromptLayouts(layout); err != nil {
		return nil, err
	}

	if err := decodeLayoutSegmentTables(doc, layout.Segments); err != nil {
		return nil, err
	}

	normalizeExtraSegment(layout.DebugPrompt)
	normalizeExtraSegment(layout.ValidLine)
	normalizeExtraSegment(layout.ErrorLine)

	if err := validateLayoutSegmentRefs(layout); err != nil {
		return nil, err
	}

	return layout, nil
}

func validateLayoutTopLevelKeys(doc map[string]any) error {
	for key, value := range doc {
		if knownLayoutTopLevelKeys[key] {
			continue
		}

		table, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("unknown top-level key %q", key)
		}

		if hasScalarFields(table) {
			continue
		}

		if isKnownSegmentType(SegmentType(key)) {
			continue
		}

		return fmt.Errorf("unknown top-level key %q", key)
	}

	return nil
}

func resolveCursorPadding(raw *layoutRawConfig) bool {
	if raw.CursorPadding != nil {
		return *raw.CursorPadding
	}

	return true
}

func (cfg *LayoutConfig) ApplyMetadata(target *Config) {
	if target == nil || cfg == nil {
		return
	}

	target.Palette = cfg.Palette
	target.Var = cfg.Var
	target.Palettes = cfg.Palettes
	target.Maps = cfg.Maps
	target.Upgrade = cfg.Upgrade
	target.Cycle = cfg.Cycle
	target.ITermFeatures = cfg.ITermFeatures
	target.VimMode = cfg.VimMode
	target.AccentColor = cfg.AccentColor
	target.DaemonIdleTimeout = cfg.DaemonIdleTimeout
	target.RenderPendingIcon = cfg.RenderPendingIcon
	target.RenderPendingBackground = cfg.RenderPendingBackground
	target.ConsoleTitleTemplate = cfg.ConsoleTitleTemplate
	target.PWD = cfg.PWD
	target.TerminalBackground = cfg.TerminalBackground
	target.ToolTipsAction = cfg.ToolTipsAction
	target.Tooltips = cfg.Tooltips
	target.DebugPrompt = cfg.DebugPrompt
	target.ValidLine = cfg.ValidLine
	target.ErrorLine = cfg.ErrorLine
	target.DaemonTimeout = cfg.DaemonTimeout
	target.Async = cfg.Async
	target.ShellIntegration = cfg.ShellIntegration
	target.CursorPadding = cfg.CursorPadding
	target.PatchPwshBleed = cfg.PatchPwshBleed
	target.EnableCursorPositioning = cfg.EnableCursorPositioning

	if len(cfg.SecondaryPrompt) > 0 {
		target.HasSecondary = true
	}

	if len(cfg.TransientPrompt) > 0 {
		target.HasTransient = true
	}
}

type separatorPair struct {
	left  string
	right string
}

var separatorAliases = map[string]separatorPair{
	"powerline":      {left: "\uE0B2", right: "\uE0B0"},
	"powerline_thin": {left: "\uE0B3", right: "\uE0B1"},
	"rounded":        {left: "\uE0B6", right: "\uE0B4"},
	"rounded_thin":   {left: "\uE0B7", right: "\uE0B5"},
	"slant":          {left: "\uE0BA", right: "\uE0BC"},
	"block":          {left: "\uE0B8", right: "\uE0BE"},
	"flame":          {left: "\uE0C0", right: "\uE0C1"},
	"pixel":          {left: "\uE0C6", right: "\uE0C6"},
	"lego":           {left: "\uE0CE", right: "\uE0CF"},
}

func normalizePromptLayouts(cfg *LayoutConfig) error {
	for i := range cfg.Prompt {
		if err := normalizePromptLayout(&cfg.Prompt[i], false, "prompt"); err != nil {
			return err
		}
	}

	for i := range cfg.RPrompt {
		if err := normalizePromptLayout(&cfg.RPrompt[i], true, "rprompt"); err != nil {
			return err
		}
	}

	for i := range cfg.SecondaryPrompt {
		if err := normalizePromptLayout(&cfg.SecondaryPrompt[i], false, "secondary"); err != nil {
			return err
		}
	}

	for i := range cfg.TransientPrompt {
		if err := normalizePromptLayout(&cfg.TransientPrompt[i], false, "transient"); err != nil {
			return err
		}
	}

	for i := range cfg.TransientRPrompt {
		if err := normalizePromptLayout(&cfg.TransientRPrompt[i], true, "rtransient"); err != nil {
			return err
		}
	}

	return nil
}

func normalizePromptLayout(layout *PromptLayout, rightAligned bool, table string) error {
	if layout.LeadingDiamond != "" || layout.TrailingDiamond != "" {
		return fmt.Errorf("%s does not allow leading_diamond/trailing_diamond", table)
	}

	if layout.LeadingStyle != "" && layout.LeadingSeparator != "" {
		return fmt.Errorf("%s cannot define both leading_style and leading_separator", table)
	}

	if layout.TrailingStyle != "" && layout.TrailingSeparator != "" {
		return fmt.Errorf("%s cannot define both trailing_style and trailing_separator", table)
	}

	if layout.Style != "" {
		if layout.LeadingStyle != "" || layout.TrailingStyle != "" || layout.LeadingSeparator != "" || layout.TrailingSeparator != "" {
			return fmt.Errorf("%s cannot define style together with explicit leading/trailing separator settings", table)
		}

		if !isSeparatorAlias(layout.Style) {
			return fmt.Errorf("%s uses unknown style alias %q", table, layout.Style)
		}

		// Shortcut behavior:
		// - left aligned lines use trailing separators
		// - right aligned lines use leading separators
		if rightAligned {
			layout.LeadingStyle = layout.Style
		} else {
			layout.TrailingStyle = layout.Style
		}
	}

	leading, err := resolveSeparator(layout.LeadingStyle, layout.LeadingSeparator, true)
	if err != nil {
		return fmt.Errorf("%s leading separator: %w", table, err)
	}

	trailing, err := resolveSeparator(layout.TrailingStyle, layout.TrailingSeparator, false)
	if err != nil {
		return fmt.Errorf("%s trailing separator: %w", table, err)
	}

	layout.LeadingDiamond = leading
	layout.TrailingDiamond = trailing
	layout.Style = ""
	layout.LeadingStyle = ""
	layout.TrailingStyle = ""
	layout.LeadingSeparator = ""
	layout.TrailingSeparator = ""

	return nil
}

func decodeLayoutSegmentTables(doc map[string]any, segmentsByName map[string]*Segment) error {
	lineTables := map[string]bool{
		"prompt":     true,
		"rprompt":    true,
		"secondary":  true,
		"transient":  true,
		"rtransient": true,
	}
	reservedTables := map[string]bool{
		"vim-mode": true,
	}

	for key, value := range doc {
		if lineTables[key] {
			continue
		}

		if reservedTables[key] {
			continue
		}

		table, ok := value.(map[string]any)
		if !ok {
			continue
		}

		if hasScalarFields(table) {
			if shouldSkipLayoutTable(key, table) {
				continue
			}

			if err := decodeLayoutSegmentTable(key, table, "", segmentsByName); err != nil {
				return err
			}
			continue
		}

		if !isKnownSegmentType(SegmentType(key)) {
			continue
		}

		for nestedKey, nestedValue := range table {
			nestedTable, ok := nestedValue.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid nested segment table")
			}

			name := fmt.Sprintf("%s.%s", key, nestedKey)
			if err := decodeLayoutSegmentTable(name, nestedTable, SegmentType(key), segmentsByName); err != nil {
				return err
			}
		}
	}

	return nil
}

func decodeLayoutSegmentTable(name string, raw map[string]any, defaultType SegmentType, segmentsByName map[string]*Segment) error {
	if _, exists := segmentsByName[name]; exists {
		return fmt.Errorf("duplicate segment instance: %s", name)
	}

	copyMap := make(map[string]any, len(raw)+1)
	maps.Copy(copyMap, raw)

	if err := normalizeSegmentSeparators(copyMap, name); err != nil {
		return err
	}

	if _, ok := copyMap["type"]; !ok {
		if defaultType != "" {
			copyMap["type"] = string(defaultType)
		} else {
			if isKnownSegmentType(SegmentType(name)) {
				copyMap["type"] = name
			}

			if _, exists := copyMap["type"]; !exists {
				if idx := strings.Index(name, "."); idx > 0 {
					candidateType := SegmentType(name[:idx])
					if isKnownSegmentType(candidateType) {
						copyMap["type"] = string(candidateType)
					}
				}
			}

			if _, exists := copyMap["type"]; !exists {
				return fmt.Errorf("segment %s is missing type", name)
			}
		}
	}

	yamlData, err := yaml.Marshal(copyMap)
	if err != nil {
		return err
	}

	var segment Segment
	if err := yaml.Unmarshal(yamlData, &segment); err != nil {
		return err
	}

	if !isKnownSegmentType(segment.Type) {
		return fmt.Errorf("unsupported segment type %q for %s", segment.Type, name)
	}

	if segment.Alias == "" {
		segment.Alias = name
	}

	segmentsByName[name] = &segment

	return nil
}

func normalizeSegmentSeparators(raw map[string]any, name string) error {
	if val, ok := raw["leading_diamond"]; ok && val != nil {
		return fmt.Errorf("%s does not allow leading_diamond", name)
	}

	if val, ok := raw["trailing_diamond"]; ok && val != nil {
		return fmt.Errorf("%s does not allow trailing_diamond", name)
	}

	leadingStyle, _ := raw["leading_style"].(string)
	leadingSeparator, _ := raw["leading_separator"].(string)
	trailingStyle, _ := raw["trailing_style"].(string)
	trailingSeparator, _ := raw["trailing_separator"].(string)
	styleValue, _ := raw["style"].(string)

	if leadingStyle != "" && leadingSeparator != "" {
		return fmt.Errorf("%s cannot define both leading_style and leading_separator", name)
	}

	if trailingStyle != "" && trailingSeparator != "" {
		return fmt.Errorf("%s cannot define both trailing_style and trailing_separator", name)
	}

	if isSeparatorAlias(styleValue) {
		if leadingStyle != "" || trailingStyle != "" || leadingSeparator != "" || trailingSeparator != "" {
			return fmt.Errorf("%s cannot define style together with explicit leading/trailing separator settings", name)
		}

		// Shortcut behavior: style defines trailing separator style, leading remains flat.
		trailingStyle = styleValue
		// Alias styles compile to diamond rendering with normalized separators.
		raw["style"] = string(Diamond)
	}

	leading, err := resolveSeparator(leadingStyle, leadingSeparator, true)
	if err != nil {
		return fmt.Errorf("%s leading separator: %w", name, err)
	}

	trailing, err := resolveSeparator(trailingStyle, trailingSeparator, false)
	if err != nil {
		return fmt.Errorf("%s trailing separator: %w", name, err)
	}

	if leading != "" {
		raw["leading_diamond"] = leading
	}

	if trailing != "" {
		raw["trailing_diamond"] = trailing
	}

	delete(raw, "leading_style")
	delete(raw, "leading_separator")
	delete(raw, "trailing_style")
	delete(raw, "trailing_separator")

	return nil
}

func resolveSeparator(style, separator string, leading bool) (string, error) {
	if separator != "" {
		return separator, nil
	}

	if style == "" {
		return "", nil
	}

	pair, ok := separatorAliases[strings.ToLower(style)]
	if !ok {
		return "", fmt.Errorf("unknown style alias %q", style)
	}

	leftGlyph := pair.left
	rightGlyph := pair.right

	if leading {
		return leftGlyph, nil
	}

	return rightGlyph, nil
}

func isSeparatorAlias(style string) bool {
	if style == "" {
		return false
	}

	_, ok := separatorAliases[strings.ToLower(style)]
	return ok
}

func hasScalarFields(table map[string]any) bool {
	for _, value := range table {
		if _, ok := value.(map[string]any); !ok {
			return true
		}
	}

	return false
}

func shouldSkipLayoutTable(name string, table map[string]any) bool {
	if _, explicitType := table["type"]; explicitType {
		return false
	}

	if isKnownSegmentType(SegmentType(name)) {
		return false
	}

	if idx := strings.Index(name, "."); idx > 0 {
		prefix := SegmentType(name[:idx])
		if isKnownSegmentType(prefix) {
			return false
		}
	}

	metadataTables := map[string]bool{
		"palette":        true,
		"palettes":       true,
		"maps":           true,
		"upgrade":        true,
		"var":            true,
		"cycle":          true,
		"iterm_features": true,
		"debug_prompt":   true,
		"valid_line":     true,
		"error_line":     true,
	}

	return metadataTables[name]
}

func normalizeExtraSegment(segment *Segment) {
	if segment == nil {
		return
	}

	if segment.Type == "" {
		segment.Type = TEXT
	}
}

func isKnownSegmentType(segmentType SegmentType) bool {
	_, ok := Segments[segmentType]
	return ok
}

func validateLayoutSegmentRefs(cfg *LayoutConfig) error {
	lines := [][]PromptLayout{
		cfg.Prompt,
		cfg.RPrompt,
		cfg.SecondaryPrompt,
		cfg.TransientPrompt,
		cfg.TransientRPrompt,
	}

	for _, lineGroup := range lines {
		for i := range lineGroup {
			for _, segmentName := range lineGroup[i].Segments {
				if _, ok := cfg.Segments[segmentName]; ok {
					continue
				}

				return fmt.Errorf("prompt references unknown segment %q", segmentName)
			}
		}
	}

	return nil
}
