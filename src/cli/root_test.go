package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCommandDoesNotExposeDeprecatedInitFlags(t *testing.T) {
	assert.Nil(t, RootCmd.Flags().Lookup("init"))
	assert.Nil(t, RootCmd.Flags().Lookup("shell"))
}
