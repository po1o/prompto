package prompt

import (
	"fmt"

	"github.com/po1o/prompto/src/log"
	"github.com/po1o/prompto/src/text"
)

func (e *Engine) Preview() string {
	builder := text.NewBuilder()

	printPrompt := func(title, prompt string) {
		builder.WriteString(log.Text(fmt.Sprintf("\n%s:\n\n", title)).Bold().Plain().String())
		builder.WriteString(prompt)
		builder.WriteString("\n")
	}

	printPrompt("Primary", e.Primary())

	right := e.RPrompt()
	if len(right) > 0 {
		printPrompt("Right", right)
	}

	if e.hasLayoutSecondary() {
		printPrompt("Secondary", e.ExtraPrompt(Secondary))
	}

	if e.hasLayoutTransient() {
		printPrompt("Transient", e.ExtraPrompt(Transient))
	}

	builder.WriteString("\n")

	return builder.String()
}
