package segments

import (
	"path/filepath"
)

type Angular struct {
	Language
}

func (a *Angular) Template() string {
	return languageTemplate
}

func (a *Angular) Enabled() bool {
	a.extensions = []string{"angular.json"}
	a.tooling = map[string]*cmd{
		"angular": {
			regex:      versionRegexNonCapturing,
			getVersion: a.getVersion,
		},
	}
	a.defaultTooling = []string{"angular"}
	a.versionURLTemplate = "https://github.com/angular/angular/releases/tag/{{.Full}}"

	return a.Language.Enabled()
}

func (a *Angular) getVersion() (string, error) {
	return a.nodePackageVersion(filepath.Join("@angular", "core"))
}
