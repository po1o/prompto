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
	name = normalizeName(name)

	content, existed := bundledConsoleThemes[name]
	delete(bundledConsoleThemes, name)

	defer func() {
		if existed {
			bundledConsoleThemes[name] = content
		}
	}()

	fn()
}

func lookup(themes map[string]string, name string) (string, bool) {
	name = normalizeName(name)
	if name == "" {
		return "", false
	}

	for key, content := range themes {
		if !strings.EqualFold(key, name) {
			continue
		}

		return content, true
	}

	return "", false
}

func normalizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, fileSuffix)
	return name
}
