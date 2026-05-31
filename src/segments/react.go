package segments

type React struct {
	Language
}

func (r *React) Template() string {
	return languageTemplate
}

func (r *React) Enabled() bool {
	r.extensions = []string{"package.json"}
	r.tooling = map[string]*cmd{
		"react": {
			regex:      versionRegexNonCapturing,
			getVersion: r.getVersion,
		},
	}
	r.defaultTooling = []string{"react"}
	r.versionURLTemplate = "https://github.com/facebook/react/releases/tag/v{{.Full}}"

	if !r.hasNodePackage("react") {
		return false
	}

	return r.Language.Enabled()
}

func (r *React) getVersion() (string, error) {
	return r.nodePackageVersion("react")
}
