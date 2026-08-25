package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/po1o/prompto/src/log"
	"github.com/po1o/prompto/src/runtime/path"
)

const (
	// ConsoleEnv forces console detection on ("1") or off ("0"), taking
	// precedence over TERM.
	ConsoleEnv = "PROMPTO_CONSOLE"
	// consoleTerm is the TERM value reported by the Linux virtual console.
	consoleTerm = "linux"
	// consoleMarker is inserted before the extension to name the console
	// variant of a config, e.g. config.yaml -> config.console.yaml.
	consoleMarker = ".console"
)

// IsConsole reports whether the prompt is rendered on a text console rather
// than a graphical terminal emulator. A console has no Nerd Font glyphs and
// only the 16 ANSI colors, so it needs a config written for those limits.
//
// PROMPTO_CONSOLE overrides TERM so the behavior can be forced either way,
// both for testing and for consoles reporting an unusual TERM.
func IsConsole(getenv func(string) string) bool {
	switch getenv(ConsoleEnv) {
	case "1":
		return true
	case "0":
		return false
	}

	return getenv("TERM") == consoleTerm
}

// ConsoleVariant returns the console-specific sibling of configFile by
// inserting ".console" before the extension:
//
//	~/.config/prompto/config.yaml -> ~/.config/prompto/config.console.yaml
func ConsoleVariant(configFile string) string {
	ext := filepath.Ext(configFile)
	return strings.TrimSuffix(configFile, ext) + consoleMarker + ext
}

// isConsoleVariant reports whether configFile already carries the console
// marker, so we never look for config.console.console.yaml.
func isConsoleVariant(configFile string) bool {
	ext := filepath.Ext(configFile)
	return strings.HasSuffix(strings.TrimSuffix(configFile, ext), consoleMarker)
}

// Resolve picks the config a session should use: the requested file, or
// DefaultPath when nothing was requested, swapped for its console variant when
// running on a console and that variant exists on disk.
//
// This runs in the shell's own process during `prompto init`, where TERM is
// the session's, and the result is baked into the init script. Every later
// render in that session then passes the resolved path back via --config, so
// console and non-console sessions can share one daemon without interfering.
func Resolve(configFile string) string {
	if configFile == "" {
		configFile = DefaultPath()
	}

	if !IsConsole(os.Getenv) {
		return configFile
	}

	expanded := path.ReplaceTildePrefixWithHomeDir(configFile)
	if isConsoleVariant(expanded) {
		return configFile
	}

	variant := ConsoleVariant(expanded)
	if _, err := os.Stat(variant); err != nil {
		log.Debugf("console detected, but no config at %s, keeping %s", variant, configFile)
		return configFile
	}

	log.Debugf("console detected, using %s", variant)

	return variant
}
