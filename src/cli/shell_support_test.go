package cli

import (
	"testing"

	"github.com/po1o/prompto/src/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveInitShellUsesExplicitValue(t *testing.T) {
	t.Setenv("PROMPTO_SHELL", "")

	sh, err := resolveInitShell("powershell")
	require.NoError(t, err)
	assert.Equal(t, shell.PWSH, sh)
}

func TestResolveInitShellUsesPromptoShellEnvWhenNoArg(t *testing.T) {
	t.Setenv("PROMPTO_SHELL", shell.ZSH)

	sh, err := resolveInitShell("")
	require.NoError(t, err)
	assert.Equal(t, shell.ZSH, sh)
}

func TestResolveInitShellRejectsUnsupportedDetectedShell(t *testing.T) {
	t.Setenv("PROMPTO_SHELL", "nu")

	_, err := resolveInitShell("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `detected "nu"`)
}
