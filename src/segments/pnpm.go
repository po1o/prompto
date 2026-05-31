package segments

type Pnpm struct {
	Language
}

func (n *Pnpm) Enabled() bool {
	n.extensions = []string{"package.json", "pnpm-lock.yaml"}
	n.tooling = map[string]*cmd{
		"pnpm": {
			executable: "pnpm",
			args:       []string{versionFlag},
			regex:      versionRegex,
		},
	}
	n.defaultTooling = []string{"pnpm"}
	n.versionURLTemplate = "https://github.com/pnpm/pnpm/releases/tag/v{{ .Full }}"

	return n.Language.Enabled()
}

func (n *Pnpm) Template() string {
	return " \ue865 {{.Full}} "
}
