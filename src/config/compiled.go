package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

type PromptLayout struct {
	Segments        []string `json:"segments,omitempty" toml:"segments,omitempty"`
	Filler          string   `json:"filler,omitempty" toml:"filler,omitempty"`
	LeadingDiamond  string   `json:"leading_diamond,omitempty" toml:"leading_diamond,omitempty"`
	TrailingDiamond string   `json:"trailing_diamond,omitempty" toml:"trailing_diamond,omitempty"`
}

type CompiledConfig struct {
	Source           string              `json:"-" toml:"-"`
	Prompt           []PromptLayout      `json:"prompt,omitempty" toml:"prompt,omitempty"`
	RPrompt          []PromptLayout      `json:"rprompt,omitempty" toml:"rprompt,omitempty"`
	SecondaryPrompt  []PromptLayout      `json:"secondary_prompt,omitempty" toml:"secondary_prompt,omitempty"`
	TransientPrompt  []PromptLayout      `json:"transient_prompt,omitempty" toml:"transient_prompt,omitempty"`
	TransientRPrompt []PromptLayout      `json:"transient_rprompt,omitempty" toml:"transient_rprompt,omitempty"`
	Segments         map[string]*Segment `json:"-" toml:"-"`
}

type compiledRawConfig struct {
	Prompt           []PromptLayout `toml:"prompt"`
	RPrompt          []PromptLayout `toml:"rprompt"`
	SecondaryPrompt  []PromptLayout `toml:"secondary_prompt"`
	TransientPrompt  []PromptLayout `toml:"transient_prompt"`
	TransientRPrompt []PromptLayout `toml:"transient_rprompt"`
}

func LoadCompiled(configFile string) (*CompiledConfig, error) {
	if configFile == "" {
		return nil, ErrNoConfig
	}

	configFile = resolveConfigLocation(configFile)
	format := strings.TrimPrefix(filepath.Ext(configFile), ".")
	if format != TOML && format != TML {
		return nil, ErrInvalidExtension
	}

	data, err := getData(configFile)
	if err != nil {
		if strings.HasPrefix(configFile, "https://") {
			return nil, ErrURLFetch
		}
		return nil, ErrFileNotFound
	}

	cfg, err := ParseCompiledTOML(data)
	if err != nil {
		return nil, err
	}

	cfg.Source = configFile

	return cfg, nil
}

func ParseCompiledTOML(data []byte) (*CompiledConfig, error) {
	var raw compiledRawConfig
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, ErrParse
	}

	var doc map[string]any
	if err := toml.Unmarshal(data, &doc); err != nil {
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

	if err := decodeCompiledSegmentTables(doc, compiled.Segments); err != nil {
		return nil, err
	}

	if err := validateCompiledSegmentRefs(compiled); err != nil {
		return nil, err
	}

	return compiled, nil
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
	for key, value := range raw {
		copyMap[key] = value
	}

	if _, ok := copyMap["type"]; !ok {
		if defaultType == "" {
			if !isKnownSegmentType(SegmentType(name)) {
				return fmt.Errorf("segment %s is missing type", name)
			}
			copyMap["type"] = name
		}

		if defaultType != "" {
			copyMap["type"] = string(defaultType)
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
		for _, line := range lineGroup {
			for _, segmentName := range line.Segments {
				if _, ok := cfg.Segments[segmentName]; ok {
					continue
				}

				return fmt.Errorf("prompt references unknown segment %q", segmentName)
			}
		}
	}

	return nil
}
