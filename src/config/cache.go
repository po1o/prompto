package config

import "github.com/po1o/prompto/src/cache"

type Cache struct {
	Duration cache.Duration `yaml:"duration,omitempty"`
	Strategy Strategy       `yaml:"strategy,omitempty"`
}

type Strategy string

const (
	Folder  Strategy = "folder"
	Session Strategy = "session"
	Device  Strategy = "device"
)
