package segments

// Vim represents the vim mode segment
type Vim struct {
	Base

	Mode     string
	Normal   bool
	Insert   bool
	Visual   bool
	Replace  bool
	Command  bool
	Operator bool
}

// Template returns the default template for the vim segment
func (v *Vim) Template() string {
	return " {{ .Mode }} "
}

// Enabled returns true if vim mode is available
func (v *Vim) Enabled() bool {
	mode := v.env.Flags().VimMode
	if mode == "" {
		return false
	}

	v.Mode = mode
	v.Normal = mode == "normal"
	v.Insert = mode == "insert"
	v.Visual = mode == "visual"
	v.Replace = mode == "replace"
	v.Command = mode == "command"
	v.Operator = mode == "operator"

	return true
}
