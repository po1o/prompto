package segments

import (
	"path/filepath"
)

type Tauri struct {
	Language
}

func (t *Tauri) Template() string {
	return languageTemplate
}

func (t *Tauri) Enabled() bool {
	t.extensions = []string{"tauri.conf.json"}
	t.folders = []string{"src-tauri"}
	t.tooling = map[string]*cmd{
		"tauri": {
			regex:      versionRegexNonCapturing,
			getVersion: t.getVersion,
		},
	}
	t.defaultTooling = []string{"tauri"}
	t.versionURLTemplate = "https://github.com/tauri-apps/tauri/releases/tag/tauri-v{{.Full}}"

	return t.Language.Enabled()
}

func (t *Tauri) getVersion() (string, error) {
	return t.nodePackageVersion(filepath.Join("@tauri-apps", "api"))
}
