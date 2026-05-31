package segments

type Deno struct {
	Language
}

func (d *Deno) Template() string {
	return languageTemplate
}

func (d *Deno) Enabled() bool {
	d.extensions = []string{"*.js", "*.ts", "deno.json"}
	d.tooling = map[string]*cmd{
		"deno": {
			executable: "deno",
			args:       []string{versionFlag},
			regex:      versionRegexNonCapturing,
		},
	}
	d.defaultTooling = []string{"deno"}
	d.versionURLTemplate = "https://github.com/denoland/deno/releases/tag/v{{.Full}}"

	return d.Language.Enabled()
}
