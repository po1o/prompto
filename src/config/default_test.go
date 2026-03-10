package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultUsesMinimalFallbackLayout(t *testing.T) {
	t.Parallel()

	cfg := Default(ErrNoConfig)
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.Layout)
	require.Len(t, cfg.Layout.Prompt, 1)
	require.Len(t, cfg.Layout.RPrompt, 1)
	assert.Equal(t, []string{"path"}, cfg.Layout.Prompt[0].Segments)
	assert.Equal(t, []string{"status"}, cfg.Layout.RPrompt[0].Segments)

	path := cfg.Layout.Segments["path"]
	require.NotNil(t, path)
	assert.Equal(t, Plain, path.Style)
	assert.Equal(t, "transparent", path.Background.String())
	assert.Empty(t, path.Foreground.String())
	assert.Equal(t, " {{ path .Path .Location }} \ue0b1", path.Template)

	status := cfg.Layout.Segments["status"]
	require.NotNil(t, status)
	assert.Equal(t, Diamond, status.Style)
	assert.Equal(t, "\ue0b6", status.LeadingDiamond)
	assert.Equal(t, "\ue0b4", status.TrailingDiamond)
	assert.Equal(t, "p:red", status.Background.String())
	assert.Equal(t, "p:white", status.Foreground.String())
}

func TestDefaultUsesConfigNotFoundMessageWhenConfigMissing(t *testing.T) {
	t.Parallel()

	assert.Equal(t, " CONFIG NOT FOUND ", Default(ErrNoConfig).Layout.Segments["status"].Template)
	assert.Equal(t, " CONFIG NOT FOUND ", Default(ErrFileNotFound).Layout.Segments["status"].Template)
}

func TestDefaultUsesConfigErrorMessageWhenConfigInvalid(t *testing.T) {
	t.Parallel()

	assert.Equal(t, " CONFIG ERROR ", Default(ErrParse).Layout.Segments["status"].Template)
	assert.Equal(t, " CONFIG ERROR ", Default(ErrInvalidExtension).Layout.Segments["status"].Template)
}
