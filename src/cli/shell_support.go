package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/po1o/prompto/src/runtime"
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

func resolveInitShell(value string) (string, error) {
	if strings.TrimSpace(value) != "" {
		return normalizeSupportedShell(value)
	}

	flags := &runtime.Flags{
		Shell: os.Getenv("PROMPTO_SHELL"),
	}

	env := &runtime.Terminal{}
	env.Init(flags)

	detected := env.Shell()
	sh, err := normalizeSupportedShell(detected)
	if err != nil {
		return "", fmt.Errorf("could not detect a supported shell automatically (detected %q): %w", detected, err)
	}

	return sh, nil
}
