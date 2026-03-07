package cli

import (
	"fmt"
	"strings"

	"github.com/po1o/prompto/src/shell"
)

var supportedShells = []string{
	shell.BASH,
	shell.ZSH,
	shell.FISH,
	"powershell",
	shell.PWSH,
}

func normalizeSupportedShell(value string) (string, error) {
	sh := strings.ToLower(strings.TrimSpace(value))
	if sh == "powershell" {
		return shell.PWSH, nil
	}

	switch sh {
	case shell.BASH, shell.ZSH, shell.FISH, shell.PWSH:
		return sh, nil
	default:
		return "", fmt.Errorf("unsupported shell %q (supported: bash, zsh, fish, powershell)", value)
	}
}
