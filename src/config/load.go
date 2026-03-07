package config

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	runtimelib "runtime"
	"strings"
	"time"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/cli/upgrade"
	"github.com/po1o/prompto/src/log"
	"github.com/po1o/prompto/src/runtime/path"
)

// Custom error types for config validation
type Error struct {
	message string
}

func (e Error) Error() string {
	return fmt.Sprintf(" %s ", e.message)
}

var (
	ErrFileNotFound     = Error{"CONFIG NOT FOUND"}
	ErrInvalidExtension = Error{"INVALID CONFIG EXTENSION"}
	ErrParse            = Error{"CONFIG PARSE ERROR"}
	ErrNoConfig         = Error{"NO CONFIG"}
)

const windowsOS = "windows"

func Load(configFile string) *Config {
	defer log.Trace(time.Now())

	cfg, err := Parse(configFile)
	if err != nil {
		cfg = Default(err)
	}

	return cfg
}

func DefaultPath() string {
	if runtimelib.GOOS == windowsOS {
		if userConfigDir, err := os.UserConfigDir(); err == nil && userConfigDir != "" {
			return filepath.Join(userConfigDir, "prompto", "config.yaml")
		}
	}

	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "prompto", "config.yaml")
	}

	home := path.Home()
	if home == "" {
		return filepath.Join(".config", "prompto", "config.yaml")
	}

	return filepath.Join(home, ".config", "prompto", "config.yaml")
}

func resolveConfigLocation(config string) string {
	defer log.Trace(time.Now())

	// Clean the config path so it works regardless of the OS
	config = filepath.ToSlash(config)

	// Cygwin path always needs the full path as we're on Windows but not really.
	// Doing filepath actions will convert it to a Windows path and break the init script.
	if isCygwin() {
		log.Debug("cygwin detected, using full path for config")
		return config
	}

	configFile := path.ReplaceTildePrefixWithHomeDir(config)

	abs, err := filepath.Abs(configFile)
	if err != nil {
		log.Error(err)
		return filepath.Clean(configFile)
	}

	return abs
}

func Parse(configFile string) (*Config, error) {
	defer log.Trace(time.Now())

	if configFile == "" {
		log.Debug("no config file specified")
		return nil, ErrNoConfig
	}

	configFile = resolveConfigLocation(configFile)

	configDSC := DSC()
	configDSC.Load()
	configDSC.Add(configFile)

	defer configDSC.Save()

	h := fnv.New64a()
	format := strings.TrimPrefix(filepath.Ext(configFile), ".")
	if format == YML {
		format = YAML
	}

	if format != YAML {
		log.Errorf("unsupported config file format: %s", format)
		return nil, ErrInvalidExtension
	}

	data, err := getData(configFile)
	if err != nil {
		log.Errorf("failed to read config: %v", err)
		return nil, ErrFileNotFound
	}

	layout, err := ParseLayoutYAML(data)
	if err != nil {
		log.Errorf("failed to parse layout config: %v", err)
		return nil, ErrParse
	}

	_, err = h.Write(data)
	if err != nil {
		log.Error(err)
	}

	cfg := Default(nil)
	cfg.Blocks = nil
	cfg.FilePaths = []string{configFile}
	cfg.Format = format
	cfg.Source = configFile
	cfg.hash = h.Sum64()
	cfg.Layout = layout

	layout.Source = configFile
	layout.ApplyMetadata(cfg)
	cfg.toggleSegments()

	if len(layout.Prompt) > 0 {
		cfg.Blocks = append(cfg.Blocks, &Block{Type: Prompt})
	}

	if len(layout.RPrompt) > 0 {
		cfg.Blocks = append(cfg.Blocks, &Block{Type: RPrompt})
	}

	if cfg.Upgrade == nil {
		cfg.Upgrade = &upgrade.Config{
			Source:        upgrade.CDN,
			DisplayNotice: cfg.UpgradeNotice,
			Auto:          cfg.AutoUpgrade,
			Interval:      cache.ONEWEEK,
		}
	}

	if cfg.Upgrade.Interval.IsEmpty() {
		cfg.Upgrade.Interval = cache.ONEWEEK
	}

	return cfg, nil
}

func getData(configFile string) ([]byte, error) {
	return os.ReadFile(configFile)
}

// isCygwin checks if we're running in Cygwin environment
func isCygwin() bool {
	return runtimelib.GOOS == "windows" && len(os.Getenv("OSTYPE")) > 0
}
