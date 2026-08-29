package segments

import "strings"

// The vim modes a shell reports through --vim-mode. The daemon precomputes the
// prompt for some of these so a mode change needs no render; see ARCHITECTURE.md.
const (
	VimInsert  = "insert"
	VimNormal  = "normal"
	VimVisual  = "visual"
	VimReplace = "replace"
)

type Vim struct {
	Base
	Insert  bool
	Normal  bool
	Visual  bool
	Replace bool
}

func (v *Vim) Enabled() bool {
	mode := strings.ToLower(v.env.Flags().VimMode)
	v.Insert = mode == VimInsert
	v.Normal = mode == VimNormal
	v.Visual = mode == VimVisual
	v.Replace = mode == VimReplace

	return true
}

func (v *Vim) Template() string {
	return "{{ if .Insert }} INSERT {{ end }}{{ if .Normal }} NORMAL {{ end }}{{ if .Visual }} VISUAL {{ end }}{{ if .Replace }} REPLACE {{ end }}"
}
