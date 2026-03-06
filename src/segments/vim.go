package segments

import "strings"

type Vim struct {
	Base
	Insert  bool
	Normal  bool
	Visual  bool
	Replace bool
}

func (v *Vim) Enabled() bool {
	mode := strings.ToLower(v.env.Flags().VimMode)
	v.Insert = mode == "insert"
	v.Normal = mode == "normal"
	v.Visual = mode == "visual"
	v.Replace = mode == "replace"

	return true
}

func (v *Vim) Template() string {
	return "{{ if .Insert }} INSERT {{ end }}{{ if .Normal }} NORMAL {{ end }}{{ if .Visual }} VISUAL {{ end }}{{ if .Replace }} REPLACE {{ end }}"
}
