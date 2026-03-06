package cli

import (
	"fmt"
	"strings"

	"github.com/jandedobbeleer/oh-my-posh/src/shell"
)

var supportedShells = []string{
	shell.BASH,
	shell.ZSH,
	shell.FISH,
	"powershell",
	shell.PWSH,
	shell.NU,
}

func normalizeSupportedShell(value string) (string, error) {
	sh := strings.ToLower(strings.TrimSpace(value))
	if sh == "powershell" {
		return shell.PWSH, nil
	}

	switch sh {
	case shell.BASH, shell.ZSH, shell.FISH, shell.PWSH, shell.NU:
		return sh, nil
	default:
		return "", fmt.Errorf("unsupported shell %q (supported: bash, zsh, fish, powershell, nu)", value)
	}
}
