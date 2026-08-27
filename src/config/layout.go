package config

import (
	"fmt"
	"maps"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/po1o/prompto/src/color"
	configmaps "github.com/po1o/prompto/src/maps"
	"github.com/po1o/prompto/src/segments"
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
	LeadingGlyph      string   `yaml:"-"`
	TrailingGlyph     string   `yaml:"-"`
	Segments          []string `yaml:"segments,omitempty"`
}

type LayoutConfig struct {
	Palette                 color.Palette          `yaml:"palette,omitempty"`
	Var                     map[string]any         `yaml:"var,omitempty"`
	Palettes                *color.Palettes        `yaml:"palettes,omitempty"`
	Maps                    *configmaps.Config     `yaml:"maps,omitempty"`
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

// removedLayoutTopLevelKeys are keys this format used to carry. They are
// rejected by name because a segment table can be named anything: without this
// an old `blocks:` is taken for a segment and reported as one missing a type,
// which points the user at the wrong problem entirely.
var removedLayoutTopLevelKeys = map[string]string{
	"upgrade":           "",
	"blocks":            "use prompt, rprompt, secondary, transient or rtransient",
	"version":           "",
	"extends":           "",
	"secondary_prompt":  "use secondary",
	"transient_prompt":  "use transient",
	"transient_rprompt": "use rtransient",
}

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

	if err := rejectLineGlyphKeys(doc); err != nil {
		return nil, err
	}

	cursorPadding := resolveCursorPadding(&raw)

	layout := &LayoutConfig{
		Palette:                 raw.Palette,
		Var:                     raw.Var,
		Palettes:                raw.Palettes,
		Maps:                    raw.Maps,
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

	if err := normalizeExtraSegments(doc, layout); err != nil {
		return nil, err
	}

	if err := validateLayoutSegmentRefs(layout); err != nil {
		return nil, err
	}

	return layout, nil
}

func validateLayoutTopLevelKeys(doc map[string]any) error {
	for key, value := range doc {
		if replacement, removed := removedLayoutTopLevelKeys[key]; removed {
			if replacement == "" {
				return fmt.Errorf("top-level key %q no longer exists", key)
			}

			return fmt.Errorf("top-level key %q no longer exists, %s", key, replacement)
		}

		if knownLayoutTopLevelKeys[key] {
			continue
		}

		table, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("unknown top-level key %q", key)
		}

		if describesASegment(table) {
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

// layoutTableNames are the top-level keys holding prompt lines.
var layoutTableNames = map[string]bool{
	"prompt":     true,
	"rprompt":    true,
	"secondary":  true,
	"transient":  true,
	"rtransient": true,
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

// mirroredGlyphs pairs each separator glyph with the one facing the other way.
// Derived from separatorAliases rather than written out again, so an alias
// cannot be added in one place and forgotten in the other.
var mirroredGlyphs = func() map[string]string {
	mirrored := make(map[string]string, len(separatorAliases)*2)

	for _, pair := range separatorAliases {
		mirrored[pair.left] = pair.right
		mirrored[pair.right] = pair.left
	}

	return mirrored
}()

// MirrorGlyph returns the glyph that faces the other way, so a separator
// written for a left-aligned line reads correctly on a right-aligned one. A
// glyph with no mirror — a custom one, or a symmetric alias like pixel — is
// returned unchanged.
func MirrorGlyph(glyph string) string {
	if mirrored, ok := mirroredGlyphs[glyph]; ok {
		return mirrored
	}

	return glyph
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

// rejectLineGlyphKeys keeps the compiled glyph fields out of user YAML. They
// are what the parser produces from style and separator keys, so accepting them
// as input would let a config bypass alias resolution and alignment.
func rejectLineGlyphKeys(doc map[string]any) error {
	for table := range layoutTableNames {
		lines, ok := doc[table].([]any)
		if !ok {
			continue
		}

		for _, line := range lines {
			fields, ok := line.(map[string]any)
			if !ok {
				continue
			}

			for _, key := range []string{"leading_glyph", "trailing_glyph"} {
				if value, ok := fields[key]; ok && value != nil {
					return fmt.Errorf("%s does not allow %s, it is set by the parser", table, key)
				}
			}

			if err := rejectRemovedSeparatorKeys(fields, table); err != nil {
				return err
			}

			if _, ok := fields["keep_when_empty"]; ok {
				return fmt.Errorf("%s does not allow keep_when_empty, it belongs on a segment", table)
			}
		}
	}

	return nil
}

func normalizePromptLayout(layout *PromptLayout, rightAligned bool, table string) error {
	leading, trailing, err := resolveSeparatorPair(&separatorSpec{
		style:             layout.Style,
		leadingStyle:      layout.LeadingStyle,
		trailingStyle:     layout.TrailingStyle,
		leadingSeparator:  layout.LeadingSeparator,
		trailingSeparator: layout.TrailingSeparator,
	}, rightAligned, table)
	if err != nil {
		return err
	}

	layout.LeadingGlyph = leading
	layout.TrailingGlyph = trailing

	layout.Style = ""
	layout.LeadingStyle = ""
	layout.TrailingStyle = ""
	layout.LeadingSeparator = ""
	layout.TrailingSeparator = ""

	return nil
}

func decodeLayoutSegmentTables(doc map[string]any, segmentsByName map[string]*Segment) error {
	lineTables := layoutTableNames
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

		if describesASegment(table) {
			if shouldSkipLayoutTable(key) {
				continue
			}

			if err := rejectAmbiguousSegmentTable(key, table); err != nil {
				return err
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
				// Reached when the table carries no key the segment format
				// declares, so it was read as a group of named instances. A
				// single misspelled key on an otherwise bare segment lands
				// here, and naming it is the whole diagnostic.
				return fmt.Errorf("%s: %q is not a segment key, and %s.%s is not a segment table either",
					key, nestedKey, key, nestedKey)
			}

			name := fmt.Sprintf("%s.%s", key, nestedKey)
			if err := decodeLayoutSegmentTable(name, nestedTable, SegmentType(key), segmentsByName); err != nil {
				return err
			}
		}
	}

	return nil
}

// rejectAmbiguousSegmentTable catches a table under a segment type name that
// reads as both forms at once: some keys the segment format declares, and some
// map-valued keys that look like named instances.
//
// `git:` carrying `cache:` and `work:` is the case. It is taken for one segment
// whose cache is configured, so `git.work` silently never exists and the error
// surfaces on whichever prompt line referenced it. Naming the collision here
// points at the table that actually caused it.
func rejectAmbiguousSegmentTable(name string, table map[string]any) error {
	if !isKnownSegmentType(SegmentType(name)) {
		return nil
	}

	var instances []string

	for key, value := range table {
		if segmentFieldNames[key] {
			continue
		}

		if _, ok := value.(map[string]any); !ok {
			continue
		}

		instances = append(instances, key)
	}

	if len(instances) == 0 {
		return nil
	}

	sort.Strings(instances)

	return fmt.Errorf("%s mixes segment keys with what look like named instances (%s); "+
		"give the instances their own table, or rename one that collides with a segment key",
		name, strings.Join(instances, ", "))
}

// inferSegmentType names the type of a segment table that did not declare one.
// A table is named either for its type ("git:") or for one instance of it
// ("git.work:"), so both forms resolve without the user repeating themselves.
func inferSegmentType(name string, defaultType SegmentType) (string, error) {
	if defaultType != "" {
		return string(defaultType), nil
	}

	if isKnownSegmentType(SegmentType(name)) {
		return name, nil
	}

	// The instance part has to be non-empty: `git.` names no instance, and
	// accepting it would let a segment through that the schema rejects.
	if idx := strings.Index(name, "."); idx > 0 && idx < len(name)-1 {
		if candidate := SegmentType(name[:idx]); isKnownSegmentType(candidate) {
			return string(candidate), nil
		}
	}

	return "", fmt.Errorf("segment %s is missing type", name)
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
		segmentType, err := inferSegmentType(name, defaultType)
		if err != nil {
			return err
		}

		copyMap["type"] = segmentType
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

	if err := validateSegmentOptions(&segment, name); err != nil {
		return err
	}

	segmentsByName[name] = &segment

	return nil
}

// validateSegmentOptions takes the name separately: tooltips and the extra
// lines have no alias to fall back on, and an error naming nothing is no help.
func validateSegmentOptions(segment *Segment, name string) error {
	if segment == nil {
		return nil
	}

	if segment.Type != TIME {
		return nil
	}

	format := segment.Options.String(segments.TimeFormat, "15:04:05")
	if segments.SupportsTimeFormat(format) {
		return nil
	}

	return fmt.Errorf("segment %s uses unsupported time_format %q", name, format)
}

// removedSeparatorKeys names the keys this config format used to carry, so an
// old config fails loudly instead of rendering as something else. A value of ""
// means the key has no replacement.
var removedSeparatorKeys = map[string]string{
	"powerline_symbol":         "trailing_separator",
	"leading_powerline_symbol": "leading_separator",
	"invert_powerline":         "",
	"leading_diamond":          "leading_separator or leading_style",
	"trailing_diamond":         "trailing_separator or trailing_style",
}

// removedSeparatorKeyNames is sorted so a config carrying several removed keys
// always reports the same one, rather than whichever the map iteration reached
// first.
var removedSeparatorKeyNames = func() []string {
	names := make([]string, 0, len(removedSeparatorKeys))
	for name := range removedSeparatorKeys {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}()

func rejectRemovedSeparatorKeys(raw map[string]any, name string) error {
	for _, key := range removedSeparatorKeyNames {
		if _, ok := raw[key]; !ok {
			continue
		}

		replacement := removedSeparatorKeys[key]
		if replacement == "" {
			return fmt.Errorf("%s uses %s, which no longer exists", name, key)
		}

		return fmt.Errorf("%s uses %s, use %s instead", name, key, replacement)
	}

	return nil
}

// separatorSpec is the separator vocabulary as written in config, before it is
// compiled to glyphs.
type separatorSpec struct {
	style             string
	leadingStyle      string
	trailingStyle     string
	leadingSeparator  string
	trailingSeparator string
}

// resolveSeparatorPair validates the separator keys and compiles them to the
// oriented glyphs the engine reads. Lines and segments share it so the rules
// cannot drift apart: they did once, and the two paths reported different
// errors for the same input.
//
// style is alignment-aware on a line — left lines close with it, right lines
// open with it. A segment has no alignment of its own, so its style always sets
// the trailing separator.
func resolveSeparatorPair(spec *separatorSpec, rightAligned bool, name string) (leadingGlyph, trailingGlyph string, err error) {
	if spec.leadingStyle != "" && spec.leadingSeparator != "" {
		return "", "", fmt.Errorf("%s cannot define both leading_style and leading_separator", name)
	}

	if spec.trailingStyle != "" && spec.trailingSeparator != "" {
		return "", "", fmt.Errorf("%s cannot define both trailing_style and trailing_separator", name)
	}

	// Resolved locally rather than by rewriting spec: the caller's value is not
	// ours to change.
	leadingStyle, trailingStyle := spec.leadingStyle, spec.trailingStyle

	// The error has to name the key the user wrote, not the side the shortcut
	// happened to expand to.
	leadingKey, trailingKey := "leading_style", "trailing_style"

	if spec.style != "" {
		if leadingStyle != "" || trailingStyle != "" || spec.leadingSeparator != "" || spec.trailingSeparator != "" {
			return "", "", fmt.Errorf("%s cannot define style together with explicit leading/trailing separator settings", name)
		}

		if rightAligned {
			leadingStyle, leadingKey = spec.style, "style"
		}

		if !rightAligned {
			trailingStyle, trailingKey = spec.style, "style"
		}
	}

	leadingGlyph, err = resolveSeparator(leadingStyle, spec.leadingSeparator, true)
	if err != nil {
		return "", "", fmt.Errorf("%s %s: %w", name, leadingKey, err)
	}

	trailingGlyph, err = resolveSeparator(trailingStyle, spec.trailingSeparator, false)
	if err != nil {
		return "", "", fmt.Errorf("%s %s: %w", name, trailingKey, err)
	}

	return leadingGlyph, trailingGlyph, nil
}

// separatorString reads one separator key, rejecting a value YAML decoded as
// something other than a string. Left unchecked, `style: 1` would be dropped
// silently and the segment would render flat with no diagnostic.
func separatorString(raw map[string]any, key, name string) (string, error) {
	value, ok := raw[key]
	if !ok || value == nil {
		return "", nil
	}

	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s %s must be a string", name, key)
	}

	return text, nil
}

func normalizeSegmentSeparators(raw map[string]any, name string) error {
	if val, ok := raw["leading_glyph"]; ok && val != nil {
		return fmt.Errorf("%s does not allow leading_glyph, it is set by the parser", name)
	}

	if val, ok := raw["trailing_glyph"]; ok && val != nil {
		return fmt.Errorf("%s does not allow trailing_glyph, it is set by the parser", name)
	}

	if err := rejectRemovedSeparatorKeys(raw, name); err != nil {
		return err
	}

	var spec separatorSpec

	for _, field := range []struct {
		target *string
		key    string
	}{
		{&spec.style, "style"},
		{&spec.leadingStyle, "leading_style"},
		{&spec.trailingStyle, "trailing_style"},
		{&spec.leadingSeparator, "leading_separator"},
		{&spec.trailingSeparator, "trailing_separator"},
	} {
		value, err := separatorString(raw, field.key, name)
		if err != nil {
			return err
		}

		*field.target = value
	}

	// A segment definition can appear on lines of either alignment, so it is
	// always compiled left-aligned; the caller mirrors it when the line it lands
	// on turns out to be right-aligned.
	leading, trailing, err := resolveSeparatorPair(&spec, false, name)
	if err != nil {
		return err
	}

	delete(raw, "style")

	if leading != "" {
		raw["leading_glyph"] = leading
	}

	if trailing != "" {
		raw["trailing_glyph"] = trailing
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
		return "", fmt.Errorf("%q is not a separator alias, expected one of: %s", style, separatorAliasNames())
	}

	leftGlyph := pair.left
	rightGlyph := pair.right

	if leading {
		return leftGlyph, nil
	}

	return rightGlyph, nil
}

// separatorAliasNames lists the aliases in a stable order so the error a user
// sees names every value they could have meant.
func separatorAliasNames() string {
	names := make([]string, 0, len(separatorAliases))
	for name := range separatorAliases {
		names = append(names, name)
	}

	sort.Strings(names)

	return strings.Join(names, ", ")
}

// separatorConfigKeys are the separator keys a user writes. They are compiled
// to glyphs and deleted before the table is decoded, so the Segment struct does
// not declare them and they have to be listed here.
var separatorConfigKeys = []string{
	"style",
	"leading_style",
	"trailing_style",
	"leading_separator",
	"trailing_separator",
}

// segmentFieldNames are the YAML keys a segment table can carry: the fields the
// struct decodes, plus the keys the parser consumes before decoding. The
// removed keys count too, so a table carrying nothing but `powerline_symbol` is
// still recognised as a segment and gets the error naming its replacement.
var segmentFieldNames = func() map[string]bool {
	fields := make(map[string]bool)

	for field := range reflect.TypeFor[Segment]().Fields() {
		key, _, _ := strings.Cut(field.Tag.Get("yaml"), ",")
		if key == "" || key == "-" {
			continue
		}

		fields[key] = true
	}

	for _, key := range separatorConfigKeys {
		fields[key] = true
	}

	for key := range removedSeparatorKeys {
		fields[key] = true
	}

	// properties is the pre-rename spelling of options, accepted in
	// Segment.UnmarshalYAML.
	fields["properties"] = true
	fields["leading_glyph"] = true
	fields["trailing_glyph"] = true

	return fields
}()

// describesASegment reports that a table is one segment rather than a group of
// named instances of one type (`git:` carrying `work:` and `personal:`).
//
// Testing for a key the Segment struct declares rather than for a scalar value:
// `options` and `cache` are the only segment keys that decode to a map, so a
// segment configured with nothing but those looked exactly like an instance
// group and was parsed as one, leaving the segment itself unregistered.
func describesASegment(table map[string]any) bool {
	for key := range table {
		if segmentFieldNames[key] {
			return true
		}
	}

	return false
}

// metadataTables are reserved top-level keys. They are never segment names, so
// they must not be registered as segments even when they carry a type: the
// extra segment tables among them are decoded separately.
var metadataTables = map[string]bool{
	"palette":        true,
	"palettes":       true,
	"maps":           true,
	"var":            true,
	"cycle":          true,
	"iterm_features": true,
	"debug_prompt":   true,
	"valid_line":     true,
	"error_line":     true,
}

// shouldSkipLayoutTable reports that a top-level table is a reserved key rather
// than a segment definition. Segment names are arbitrary, so this is the only
// thing standing between `palette:` and being parsed as a segment called
// "palette".
func shouldSkipLayoutTable(name string) bool {
	return metadataTables[name]
}

// extraSegmentTables are segments the config addresses by role rather than by
// name. They are decoded from the raw document so they pass through the same
// separator normalization as named segments, which both compiles their
// separator keys and rejects the engine-level ones.
var extraSegmentTables = []string{"debug_prompt", "valid_line", "error_line"}

func normalizeExtraSegments(doc map[string]any, layout *LayoutConfig) error {
	targets := map[string]**Segment{
		"debug_prompt": &layout.DebugPrompt,
		"valid_line":   &layout.ValidLine,
		"error_line":   &layout.ErrorLine,
	}

	for _, name := range extraSegmentTables {
		table, ok := doc[name].(map[string]any)
		if !ok {
			continue
		}

		segment, err := decodeExtraSegment(table, name, TEXT)
		if err != nil {
			return err
		}

		*targets[name] = segment
	}

	tooltips, ok := doc["tooltips"].([]any)
	if !ok {
		return nil
	}

	layout.Tooltips = make([]*Segment, 0, len(tooltips))

	for i, item := range tooltips {
		table, ok := item.(map[string]any)
		if !ok {
			return fmt.Errorf("tooltip %d is not a table", i)
		}

		segment, err := decodeExtraSegment(table, fmt.Sprintf("tooltip %d", i), "")
		if err != nil {
			return err
		}

		// A tooltip only ever renders in a right-aligned block, and Tooltip()
		// builds that block directly rather than going through layoutBlock, so
		// nothing else would turn its separators around. Mirroring here is what
		// makes a tooltip carry the same separator keys as a segment named on an
		// rprompt line and get the same shape.
		segment.MirrorSeparators()

		layout.Tooltips = append(layout.Tooltips, segment)
	}

	return nil
}

func decodeExtraSegment(raw map[string]any, name string, defaultType SegmentType) (*Segment, error) {
	copyMap := make(map[string]any, len(raw)+1)
	maps.Copy(copyMap, raw)

	if err := normalizeSegmentSeparators(copyMap, name); err != nil {
		return nil, err
	}

	if _, ok := copyMap["type"]; !ok && defaultType != "" {
		copyMap["type"] = string(defaultType)
	}

	yamlData, err := yaml.Marshal(copyMap)
	if err != nil {
		return nil, err
	}

	var segment Segment
	if err := yaml.Unmarshal(yamlData, &segment); err != nil {
		return nil, err
	}

	if segment.Type == "" {
		return nil, fmt.Errorf("%s is missing type", name)
	}

	if !isKnownSegmentType(segment.Type) {
		return nil, fmt.Errorf("unsupported segment type %q for %s", segment.Type, name)
	}

	if err := validateSegmentOptions(&segment, name); err != nil {
		return nil, err
	}

	return &segment, nil
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
