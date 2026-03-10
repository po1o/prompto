package config

import (
	"github.com/po1o/prompto/src/color"
	"github.com/po1o/prompto/src/segments/options"
)

func defaultConfigErrorText(configError error) string {
	switch configError {
	case ErrNoConfig, ErrFileNotFound:
		return " CONFIG NOT FOUND "
	case nil:
		return " CONFIG ERROR "
	default:
		return " CONFIG ERROR "
	}
}

func Default(configError error) *Config {
	configErrorText := defaultConfigErrorText(configError)

	layout := &LayoutConfig{
		Prompt: []PromptLayout{
			{Segments: []string{"path"}},
		},
		RPrompt: []PromptLayout{
			{Segments: []string{"status"}},
		},
		Segments: map[string]*Segment{
			"path": {
				Type:       PATH,
				Style:      Plain,
				Background: "transparent",
				Options: options.Map{
					options.Style: "folder",
				},
				Template: " {{ path .Path .Location }} \ue0b1",
			},
			"status": {
				Type:            STATUS,
				Style:           Diamond,
				LeadingDiamond:  "\ue0b6",
				TrailingDiamond: "\ue0b4",
				Foreground:      "p:white",
				Background:      "p:red",
				Options: options.Map{
					options.AlwaysEnabled: true,
				},
				Template: configErrorText,
			},
		},
	}

	cfg := &Config{
		hash:                 1234567890, // placeholder hash value
		CursorPadding:        true,
		Layout:               layout,
		ConsoleTitleTemplate: "{{ .Shell }} in {{ .Folder }}",
		Palette: color.Palette{
			"black":  "#262B44",
			"blue":   "#4B95E9",
			"green":  "#59C9A5",
			"orange": "#F07623",
			"red":    "#D81E5B",
			"white":  "#E0DEF4",
			"yellow": "#F3AE35",
		},
		Tooltips: []*Segment{
			{
				Type:            AWS,
				Style:           Diamond,
				LeadingDiamond:  "\ue0b0",
				TrailingDiamond: "\ue0b4",
				Foreground:      "p:white",
				Background:      "p:orange",
				Template:        " \ue7ad {{ .Profile }}{{ if .Region }}@{{ .Region }}{{ end }} ",
				Options: options.Map{
					options.DisplayDefault: true,
				},
				Tips: []string{"aws"},
			},
			{
				Type:            AZ,
				Style:           Diamond,
				LeadingDiamond:  "\ue0b0",
				TrailingDiamond: "\ue0b4",
				Foreground:      "p:white",
				Background:      "p:blue",
				Template:        " \uebd8 {{ .Name }} ",
				Options: options.Map{
					options.DisplayDefault: true,
				},
				Tips: []string{"az"},
			},
		},
	}

	cfg.toggleSegments()

	return cfg
}
