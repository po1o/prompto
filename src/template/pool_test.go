package template

import (
	"testing"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/runtime/mock"

	"github.com/stretchr/testify/assert"
)

func TestTextPool(t *testing.T) {
	env := new(mock.Environment)
	env.On("Shell").Return("foo")
	Cache = new(cache.Template)
	Init(env, nil, nil)

	// Test rendering
	result, err := Render("Hello {{ .Name }}", map[string]any{"Name": "World"})
	assert.NoError(t, err)
	assert.Equal(t, "Hello World", result)

	// Test empty template
	result2, err := Render("", nil)
	assert.NoError(t, err)
	assert.Equal(t, "", result2)
}
