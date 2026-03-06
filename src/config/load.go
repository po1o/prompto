package config

import (
	"errors"
	"fmt"
	"hash"
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
	yaml "go.yaml.in/yaml/v3"
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

func Load(configFile string) *Config {
	defer log.Trace(time.Now())

	cfg, err := Parse(configFile)
	if err != nil {
		cfg = Default(err)
	}

	return cfg
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

	// Hash is used as lightweight config identity for change detection/caching.
	h := fnv.New64a()

	cfg, err := read(configFile, h)
	if err != nil {
		log.Errorf("failed to read config: %s", configFile)
		return nil, err
	}

	parentFolder := filepath.Dir(configFile)
	filePaths := []string{configFile}

	for cfg.Extends != "" {
		// Resolve relative extends from the current config directory.
		cfg.Extends = resolvePath(cfg.Extends, parentFolder)
		filePaths = append(filePaths, cfg.Extends)
		base, err := read(cfg.Extends, h)
		if err != nil {
			log.Errorf("failed to read extended config: %s", cfg.Extends)
			break
		}

		configDSC.Add(cfg.Extends)

		err = base.merge(cfg)
		if err != nil {
			log.Error(err)
			break
		}

		cfg = base
	}

	cfg.Source = configFile
	cfg.FilePaths = filePaths
	cfg.hash = h.Sum64()
	cfg.migrateSegmentProperties()

	cfg.toggleSegments()

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

func resolvePath(configFile, parentFolder string) string {
	configFile = path.ReplaceTildePrefixWithHomeDir(configFile)

	if filepath.IsAbs(configFile) {
		return configFile
	}

	return filepath.Join(parentFolder, configFile)
}

func read(configFile string, h hash.Hash64) (*Config, error) {
	defer log.Trace(time.Now())

	if configFile == "" {
		log.Debug("no config file specified, using default")
		return Default(nil), nil
	}

	var cfg Config
	cfg.Source = configFile
	format := strings.TrimPrefix(filepath.Ext(configFile), ".")
	if format == YML {
		format = YAML
	}
	cfg.Format = format

	data, err := getData(configFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Errorf("config file not found: %v", err)
			return nil, ErrFileNotFound
		}
		log.Errorf("failed to read config: %v", err)
		return nil, ErrFileNotFound
	}

	if cfg.Format != YAML {
		log.Errorf("unsupported config file format: %s", cfg.Format)
		return nil, ErrInvalidExtension
	}

	parseErr := yaml.Unmarshal(data, &cfg)
	if parseErr != nil {
		log.Errorf("failed to parse config: %v", parseErr)
		return nil, ErrParse
	}

	_, err = h.Write(data)
	if err != nil {
		log.Error(err)
	}

	return &cfg, nil
}

func getData(configFile string) ([]byte, error) {
	return os.ReadFile(configFile)
}

// isCygwin checks if we're running in Cygwin environment
func isCygwin() bool {
	return runtimelib.GOOS == "windows" && len(os.Getenv("OSTYPE")) > 0
}
