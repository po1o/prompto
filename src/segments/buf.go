package segments

type Buf struct {
	Language
}

func (b *Buf) Template() string {
	return languageTemplate
}

func (b *Buf) Enabled() bool {
	b.extensions = []string{"buf.yaml", "buf.gen.yaml", "buf.work.yaml"}
	b.tooling = map[string]*cmd{
		"buf": {
			executable: "buf",
			args:       []string{versionFlag},
			regex:      versionRegexNonCapturing,
		},
	}
	b.defaultTooling = []string{"buf"}
	b.versionURLTemplate = "https://github.com/bufbuild/buf/releases/tag/v{{.Full}}"

	return b.Language.Enabled()
}
