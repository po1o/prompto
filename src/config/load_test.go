package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/po1o/prompto/src/runtime/mock"
	"github.com/po1o/prompto/src/shell"
	"github.com/stretchr/testify/require"
)

func TestParseDoesNotMergeDefaultTooltips(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `version: 4
prompt:
  - segments:
      - path
path:
  style: powerline
  template: " {{ .Path }} "
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	cfg, err := Parse(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Empty(t, cfg.Tooltips)

	env := &mock.Environment{}
	env.On("Shell").Return(shell.ZSH)

	features := cfg.Features(env, false)
	require.Zero(t, features&shell.Tooltips)
}

func TestParseLoadsYAMLTooltips(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `version: 4
tooltips_action: replace
tooltips:
  - type: text
    tips:
      - cd
    template: " tip "
prompt:
  - segments:
      - path
path:
  style: powerline
  template: " {{ .Path }} "
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	cfg, err := Parse(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Tooltips, 1)
	require.True(t, cfg.ToolTipsAction.IsDefault())
	require.Equal(t, []string{"cd"}, cfg.Tooltips[0].Tips)

	env := &mock.Environment{}
	env.On("Shell").Return(shell.ZSH)

	features := cfg.Features(env, false)
	require.NotZero(t, features&shell.Tooltips)
}
