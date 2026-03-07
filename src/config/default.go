package config

import (
	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/cli/upgrade"
	"github.com/po1o/prompto/src/color"
	"github.com/po1o/prompto/src/segments/options"
)

func Default(configError error) *Config {
	exitBackgroundTemplate := "{{ if gt .Code 0 }}p:red{{ end }}"
	exitTemplate := " {{ if gt .Code 0 }}\uf00d{{ else }}\uf00c{{ end }} "

	if configError != nil && configError != ErrNoConfig {
		exitBackgroundTemplate = "p:red"
		exitTemplate = configError.Error()
	}

	layout := &LayoutConfig{
		Prompt: []PromptLayout{
			{Segments: []string{"session", "path", "status"}},
		},
		RPrompt: []PromptLayout{
			{Segments: []string{"shell", "time"}},
		},
		Segments: map[string]*Segment{
			"session": {
				Type:            SESSION,
				Style:           Diamond,
				LeadingDiamond:  "\ue0b6",
				TrailingDiamond: "\ue0b0",
				Foreground:      "p:black",
				Background:      "p:yellow",
				Template:        " {{ if .SSHSession }}\ueba9 {{ end }}{{ .UserName }} ",
			},
			"path": {
				Type:            PATH,
				Style:           Powerline,
				PowerlineSymbol: "\ue0b0",
				Foreground:      "p:white",
				Background:      "p:orange",
				Options: options.Map{
					options.Style: "folder",
				},
				Template: " \uea83 {{ path .Path .Location }} ",
			},
			"status": {
				Type:            STATUS,
				Style:           Diamond,
				LeadingDiamond:  "<transparent,background>\ue0b0</>",
				TrailingDiamond: "\ue0b4",
				Foreground:      "p:white",
				Background:      "p:blue",
				BackgroundTemplates: []string{
					exitBackgroundTemplate,
				},
				Options: options.Map{
					options.AlwaysEnabled: true,
				},
				Template: exitTemplate,
			},
			"shell": {
				Type:       SHELL,
				Style:      Plain,
				Foreground: "p:white",
				Background: "transparent",
				Template:   "in <p:blue><b>{{ .Name }}</b></> ",
			},
			"time": {
				Type:       TIME,
				Style:      Plain,
				Foreground: "p:white",
				Background: "transparent",
				Template:   "at <p:blue><b>{{ .CurrentDate | date \"15:04:05\" }}</b></>",
			},
		},
	}

	cfg := &Config{
		hash:                 1234567890, // placeholder hash value
		FinalSpace:           true,
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
		Upgrade: &upgrade.Config{
			Source:   upgrade.CDN,
			Interval: cache.ONEWEEK,
		},
	}

	cfg.toggleSegments()

	return cfg
}
