package shell

import (
	"strings"

	"github.com/po1o/prompto/src/text"
)

type Code string

const (
	unixFTCSMarks         Code = "_prompto_ftcs_marks=1"
	unixCursorPositioning Code = "_prompto_cursor_positioning=1"
	unixUpgrade           Code = `"$_prompto_executable" upgrade --auto`
	unixNotice            Code = `"$_prompto_executable" notice`
	enablePromptoDaemon   Code = "enable_prompto_daemon"
)

func (c Code) Indent(spaces int) Code {
	return Code(strings.Repeat(" ", spaces) + string(c))
}

type Lines []Code

func (l Lines) String(script string) string {
	builder := text.NewBuilder()

	builder.WriteString(script)
	builder.WriteString("\n")

	for i, line := range l {
		builder.WriteString(string(line))

		// add newline if not last line
		if i < len(l)-1 {
			builder.WriteString("\n")
		}
	}

	return builder.String()
}
