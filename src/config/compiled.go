package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"strings"

	"github.com/po1o/prompto/src/cli/upgrade"
	"github.com/po1o/prompto/src/color"
	configmaps "github.com/po1o/prompto/src/maps"
	"github.com/po1o/prompto/src/terminal"
	yaml "go.yaml.in/yaml/v3"
)

type PromptLayout struct {
	Style             string   `json:"style,omitempty" yaml:"style,omitempty"`
	Filler            string   `json:"filler,omitempty" yaml:"filler,omitempty"`
	LeadingStyle      string   `json:"leading_style,omitempty" yaml:"leading_style,omitempty"`
	TrailingStyle     string   `json:"trailing_style,omitempty" yaml:"trailing_style,omitempty"`
	LeadingSeparator  string   `json:"leading_separator,omitempty" yaml:"leading_separator,omitempty"`
	TrailingSeparator string   `json:"trailing_separator,omitempty" yaml:"trailing_separator,omitempty"`
	LeadingDiamond    string   `json:"leading_diamond,omitempty" yaml:"leading_diamond,omitempty"`
	TrailingDiamond   string   `json:"trailing_diamond,omitempty" yaml:"trailing_diamond,omitempty"`
	Segments          []string `json:"segments,omitempty" yaml:"segments,omitempty"`
}

type CompiledConfig struct {
	Palette          color.Palette          `json:"palette,omitempty" yaml:"palette,omitempty"`
	Var              map[string]any         `json:"var,omitempty" yaml:"var,omitempty"`
	Palettes         *color.Palettes        `json:"palettes,omitempty" yaml:"palettes,omitempty"`
	Maps             *configmaps.Config     `json:"maps,omitempty" yaml:"maps,omitempty"`
	Upgrade          *upgrade.Config        `json:"upgrade,omitempty" yaml:"upgrade,omitempty"`
	Cycle            color.Cycle            `json:"cycle,omitempty" yaml:"cycle,omitempty"`
	ITermFeatures    terminal.ITermFeatures `json:"iterm_features,omitempty" yaml:"iterm_features,omitempty"`
	VimMode          *VimConfig             `json:"vim-mode,omitempty" yaml:"vim-mode,omitempty"`
	Segments         map[string]*Segment    `json:"-" yaml:"-"`
	Source           string                 `json:"-" yaml:"-"`
	Prompt           []PromptLayout         `json:"prompt,omitempty" yaml:"prompt,omitempty"`
	RPrompt          []PromptLayout         `json:"rprompt,omitempty" yaml:"rprompt,omitempty"`
	SecondaryPrompt  []PromptLayout         `json:"secondary,omitempty" yaml:"secondary,omitempty"`
	TransientPrompt  []PromptLayout         `json:"transient,omitempty" yaml:"transient,omitempty"`
	TransientRPrompt []PromptLayout         `json:"rtransient,omitempty" yaml:"rtransient,omitempty"`
}

type compiledRawConfig struct {
	Palette       color.Palette          `yaml:"palette"`
	Var           map[string]any         `yaml:"var"`
	Palettes      *color.Palettes        `yaml:"palettes"`
	Maps          *configmaps.Config     `yaml:"maps"`
	Upgrade       *upgrade.Config        `yaml:"upgrade"`
	Cycle         color.Cycle            `yaml:"cycle"`
	ITermFeatures terminal.ITermFeatures `yaml:"iterm_features"`
	VimMode       *VimConfig             `yaml:"vim-mode"`
	Prompt        []PromptLayout         `yaml:"prompt"`
	RPrompt       []PromptLayout         `yaml:"rprompt"`
	Secondary     []PromptLayout         `yaml:"secondary"`
	Transient     []PromptLayout         `yaml:"transient"`
	RTransient    []PromptLayout         `yaml:"rtransient"`
}

func LoadCompiled(configFile string) (*CompiledConfig, error) {
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

	cfg, err := ParseCompiledYAML(data)
	if err != nil {
		return nil, err
	}

	cfg.Source = configFile

	return cfg, nil
}

func ParseCompiledYAML(data []byte) (*CompiledConfig, error) {
	var raw compiledRawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, ErrParse
	}

	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, ErrParse
	}

	compiled := &CompiledConfig{
		Palette:          raw.Palette,
		Var:              raw.Var,
		Palettes:         raw.Palettes,
		Maps:             raw.Maps,
		Upgrade:          raw.Upgrade,
		Cycle:            raw.Cycle,
		ITermFeatures:    raw.ITermFeatures,
		VimMode:          raw.VimMode,
		Prompt:           raw.Prompt,
		RPrompt:          raw.RPrompt,
		SecondaryPrompt:  raw.Secondary,
		TransientPrompt:  raw.Transient,
		TransientRPrompt: raw.RTransient,
		Segments:         make(map[string]*Segment),
	}

	if err := normalizePromptLayouts(compiled); err != nil {
		return nil, err
	}

	if err := validateCompiledTopLevelTables(doc); err != nil {
		return nil, err
	}

	if err := decodeCompiledSegmentTables(doc, compiled.Segments); err != nil {
		return nil, err
	}

	if err := validateCompiledSegmentRefs(compiled); err != nil {
		return nil, err
	}

	return compiled, nil
}

func (cfg *CompiledConfig) ApplyMetadata(target *Config) {
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

	if len(cfg.SecondaryPrompt) > 0 {
		target.SecondaryPrompt = &Segment{}
	}

	if len(cfg.TransientPrompt) > 0 {
		target.TransientPrompt = &Segment{}
	}
}

func validateCompiledTopLevelTables(doc map[string]any) error {
	if _, hasSecondaryPrompt := doc["secondary_prompt"]; hasSecondaryPrompt {
		return errors.New("top-level secondary_prompt is not supported; use secondary")
	}

	if _, hasTransientPrompt := doc["transient_prompt"]; hasTransientPrompt {
		return errors.New("top-level transient_prompt is not supported; use transient")
	}

	if _, hasTransientRPrompt := doc["transient_rprompt"]; hasTransientRPrompt {
		return errors.New("top-level transient_rprompt is not supported; use rtransient")
	}

	rawVim, ok := doc["vim"]
	if !ok {
		return nil
	}

	vimTable, ok := rawVim.(map[string]any)
	if !ok {
		return nil
	}

	if _, hasEnabled := vimTable["enabled"]; hasEnabled {
		return errors.New("top-level vim is not supported; use vim-mode")
	}

	if _, hasCursorShape := vimTable["cursor_shape"]; hasCursorShape {
		return errors.New("top-level vim is not supported; use vim-mode")
	}

	if _, hasCursorBlink := vimTable["cursor_blink"]; hasCursorBlink {
		return errors.New("top-level vim is not supported; use vim-mode")
	}

	return nil
}

type separatorPair struct {
	left  string
	right string
}

var separatorAliases = map[string]separatorPair{
	"powerline":      {left: "\uE0B2", right: "\uE0B0"},
	"powerline_thin": {left: "\uE0B3", right: "\uE0B1"},
	"rounded":        {left: "\uE0B6", right: "\uE0B4"},
	"slant":          {left: "\uE0BA", right: "\uE0BC"},
	"block":          {left: "\uE0B8", right: "\uE0BE"},
	"flame":          {left: "\uE0C0", right: "\uE0C1"},
	"pixel":          {left: "\uE0C6", right: "\uE0C6"},
	"lego":           {left: "\uE0CE", right: "\uE0CF"},
}

func normalizePromptLayouts(cfg *CompiledConfig) error {
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

func decodeCompiledSegmentTables(doc map[string]any, segmentsByName map[string]*Segment) error {
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
			if shouldSkipCompiledTable(key, table) {
				continue
			}

			if err := decodeCompiledSegmentTable(key, table, "", segmentsByName); err != nil {
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
				return errors.New("invalid nested segment table")
			}

			name := fmt.Sprintf("%s.%s", key, nestedKey)
			if err := decodeCompiledSegmentTable(name, nestedTable, SegmentType(key), segmentsByName); err != nil {
				return err
			}
		}
	}

	return nil
}

func decodeCompiledSegmentTable(name string, raw map[string]any, defaultType SegmentType, segmentsByName map[string]*Segment) error {
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

	jsonData, err := json.Marshal(copyMap)
	if err != nil {
		return err
	}

	var segment Segment
	if err := json.Unmarshal(jsonData, &segment); err != nil {
		return err
	}

	if !isKnownSegmentType(segment.Type) {
		return fmt.Errorf("unsupported segment type %q for %s", segment.Type, name)
	}

	if segment.Alias == "" {
		segment.Alias = name
	}

	segment.MigratePropertiesToOptions()
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

func shouldSkipCompiledTable(name string, table map[string]any) bool {
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
		"cache":          true,
		"var":            true,
		"cycle":          true,
		"iterm_features": true,
		"secondary":      true,
		"transient":      true,
		"valid_line":     true,
		"error_line":     true,
		"debug_prompt":   true,
	}

	return metadataTables[name]
}

func isKnownSegmentType(segmentType SegmentType) bool {
	_, ok := Segments[segmentType]
	return ok
}

func validateCompiledSegmentRefs(cfg *CompiledConfig) error {
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
