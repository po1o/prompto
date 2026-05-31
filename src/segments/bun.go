package segments

type Bun struct {
	Language
}

func (b *Bun) Template() string {
	return languageTemplate
}

func (b *Bun) Enabled() bool {
	b.extensions = []string{"bun.lockb", "bun.lock"}
	b.tooling = map[string]*cmd{
		"bun": {
			executable: "bun",
			args:       []string{versionFlag},
			regex:      versionRegexNonCapturing,
		},
	}
	b.defaultTooling = []string{"bun"}
	b.versionURLTemplate = "https://github.com/oven-sh/bun/releases/tag/bun-v{{.Full}}"

	return b.Language.Enabled()
}
