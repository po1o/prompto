package themes

import (
	"sort"
	"strings"
)

const fileSuffix = ".prompto.yaml"

//go:generate go run ../tools/genthemes

func Names() []string {
	names := make([]string, 0, len(bundledThemes))
	for name := range bundledThemes {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

func Get(name string) (string, bool) {
	return lookup(bundledThemes, name)
}

// GetConsole returns the console variant of a theme, bundled from
// <name>.console.prompto.yaml. Most themes have none, so the second return
// value reports whether one exists rather than falling back to Get: the two
// are written for different terminals and are not interchangeable.
func GetConsole(name string) (string, bool) {
	return lookup(bundledConsoleThemes, name)
}

// WithoutConsoleVariant hides a theme's console variant for the duration of fn,
// so tests can still reach the path a theme takes when it ships none. Every
// bundled theme currently has one, which would otherwise leave that path
// untested until a console-safe theme is added and silently uncovers it.
//
// It mutates package state, so tests using it must not run in parallel.
func WithoutConsoleVariant(name string, fn func()) {
	// Matched the way GetConsole matches, or a caller passing "Tokyo" would
	// delete nothing and run fn against the variant it meant to hide.
	key, existed := lookupKey(bundledConsoleThemes, name)

	content := bundledConsoleThemes[key]
	delete(bundledConsoleThemes, key)

	defer func() {
		if existed {
			bundledConsoleThemes[key] = content
		}
	}()

	fn()
}

func lookup(themes map[string]string, name string) (string, bool) {
	key, ok := lookupKey(themes, name)
	if !ok {
		return "", false
	}

	return themes[key], true
}

// lookupKey finds the stored key for name, matching case-insensitively so a
// theme can be named the way the user typed it.
func lookupKey(themes map[string]string, name string) (string, bool) {
	name = normalizeName(name)
	if name == "" {
		return "", false
	}

	for key := range themes {
		if !strings.EqualFold(key, name) {
			continue
		}

		return key, true
	}

	return "", false
}

func normalizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, fileSuffix)
	return name
}
