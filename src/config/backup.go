package config

import (
	"bytes"
	"io"
	"os"

	yaml "go.yaml.in/yaml/v3"
)

func (cfg *Config) Backup() {
	dst := cfg.Source + ".bak"
	source, err := os.Open(cfg.Source)
	if err != nil {
		return
	}
	defer source.Close()
	destination, err := os.Create(dst)
	if err != nil {
		return
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	if err != nil {
		return
	}
}

func (cfg *Config) Export(format string) string {
	if len(format) != 0 {
		cfg.Format = format
	}

	if cfg.Format == "" || cfg.Format == YML {
		cfg.Format = YAML
	}

	if cfg.Format != YAML {
		return ""
	}

	var result bytes.Buffer
	prefix := "# yaml-language-server: $schema=https://raw.githubusercontent.com/JanDeDobbeleer/oh-my-posh/main/themes/schema.json\n\n"
	yamlEncoder := yaml.NewEncoder(&result)

	err := yamlEncoder.Encode(cfg)
	if err != nil {
		return ""
	}

	return prefix + result.String()
}

func (cfg *Config) Write(format string) {
	content := cfg.Export(format)
	if content == "" {
		// we are unable to perform the export
		return
	}

	f, err := os.OpenFile(cfg.Source, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return
	}

	defer func() {
		_ = f.Close()
	}()

	_, err = f.WriteString(content)
	if err != nil {
		return
	}
}
