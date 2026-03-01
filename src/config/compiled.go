package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

type PromptLayout struct {
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
	Segments         map[string]*Segment `json:"-" yaml:"-"`
	Source           string              `json:"-" yaml:"-"`
	Prompt           []PromptLayout      `json:"prompt,omitempty" yaml:"prompt,omitempty"`
	RPrompt          []PromptLayout      `json:"rprompt,omitempty" yaml:"rprompt,omitempty"`
	SecondaryPrompt  []PromptLayout      `json:"secondary_prompt,omitempty" yaml:"secondary_prompt,omitempty"`
	TransientPrompt  []PromptLayout      `json:"transient_prompt,omitempty" yaml:"transient_prompt,omitempty"`
	TransientRPrompt []PromptLayout      `json:"transient_rprompt,omitempty" yaml:"transient_rprompt,omitempty"`
}

type compiledRawConfig struct {
	Prompt           []PromptLayout `yaml:"prompt"`
	RPrompt          []PromptLayout `yaml:"rprompt"`
	SecondaryPrompt  []PromptLayout `yaml:"secondary_prompt"`
	TransientPrompt  []PromptLayout `yaml:"transient_prompt"`
	TransientRPrompt []PromptLayout `yaml:"transient_rprompt"`
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
		if strings.HasPrefix(configFile, "https://") {
			return nil, ErrURLFetch
		}
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
		Prompt:           raw.Prompt,
		RPrompt:          raw.RPrompt,
		SecondaryPrompt:  raw.SecondaryPrompt,
		TransientPrompt:  raw.TransientPrompt,
		TransientRPrompt: raw.TransientRPrompt,
		Segments:         make(map[string]*Segment),
	}

	if err := normalizePromptLayouts(compiled); err != nil {
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
		if err := normalizePromptLayout(&cfg.SecondaryPrompt[i], false, "secondary_prompt"); err != nil {
			return err
		}
	}

	for i := range cfg.TransientPrompt {
		if err := normalizePromptLayout(&cfg.TransientPrompt[i], false, "transient_prompt"); err != nil {
			return err
		}
	}

	for i := range cfg.TransientRPrompt {
		if err := normalizePromptLayout(&cfg.TransientRPrompt[i], true, "transient_rprompt"); err != nil {
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

	leading, err := resolveSeparator(layout.LeadingStyle, layout.LeadingSeparator, rightAligned, true)
	if err != nil {
		return fmt.Errorf("%s leading separator: %w", table, err)
	}

	trailing, err := resolveSeparator(layout.TrailingStyle, layout.TrailingSeparator, rightAligned, false)
	if err != nil {
		return fmt.Errorf("%s trailing separator: %w", table, err)
	}

	layout.LeadingDiamond = leading
	layout.TrailingDiamond = trailing
	layout.LeadingStyle = ""
	layout.TrailingStyle = ""
	layout.LeadingSeparator = ""
	layout.TrailingSeparator = ""

	return nil
}

func decodeCompiledSegmentTables(doc map[string]any, segmentsByName map[string]*Segment) error {
	lineTables := map[string]bool{
		"prompt":            true,
		"rprompt":           true,
		"secondary_prompt":  true,
		"transient_prompt":  true,
		"transient_rprompt": true,
	}

	for key, value := range doc {
		if lineTables[key] {
			continue
		}

		table, ok := value.(map[string]any)
		if !ok {
			continue
		}

		if hasScalarFields(table) {
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

	if leadingStyle != "" && leadingSeparator != "" {
		return fmt.Errorf("%s cannot define both leading_style and leading_separator", name)
	}

	if trailingStyle != "" && trailingSeparator != "" {
		return fmt.Errorf("%s cannot define both trailing_style and trailing_separator", name)
	}

	leading, err := resolveSeparator(leadingStyle, leadingSeparator, false, true)
	if err != nil {
		return fmt.Errorf("%s leading separator: %w", name, err)
	}

	trailing, err := resolveSeparator(trailingStyle, trailingSeparator, false, false)
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

func resolveSeparator(style, separator string, rightAligned, leading bool) (string, error) {
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

	if rightAligned {
		leftGlyph, rightGlyph = rightGlyph, leftGlyph
	}

	if leading {
		return leftGlyph, nil
	}

	return rightGlyph, nil
}

func hasScalarFields(table map[string]any) bool {
	for _, value := range table {
		if _, ok := value.(map[string]any); !ok {
			return true
		}
	}

	return false
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
