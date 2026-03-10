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
	name = normalizeName(name)
	if name == "" {
		return "", false
	}

	for key, content := range bundledThemes {
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
