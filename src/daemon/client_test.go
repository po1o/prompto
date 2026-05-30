package daemon

import (
	"testing"

	"github.com/po1o/prompto/src/daemon/ipc"

	"github.com/stretchr/testify/require"
)

// The client's connection and RPC wrappers need an in-process gRPC harness
// to exercise (the existing server_test.go style). What is cheap to test
// here is ExtractPrompts — pure response parsing with zero side effects and
// a long history of B1 flagging it at 0% coverage.

func TestExtractPromptsNilResponse(t *testing.T) {
	result := ExtractPrompts(nil)
	require.NotNil(t, result)
	require.Equal(t, PromptResult{}, *result)
}

func TestExtractPromptsEmptyPrompts(t *testing.T) {
	result := ExtractPrompts(&ipc.PromptResponse{})
	require.NotNil(t, result)
	require.Equal(t, PromptResult{}, *result)
}

func TestExtractPromptsAllFieldsPopulated(t *testing.T) {
	result := ExtractPrompts(&ipc.PromptResponse{
		Prompts: map[string]*ipc.Prompt{
			"primary":    {Text: "P"},
			"right":      {Text: "R"},
			"secondary":  {Text: "S"},
			"transient":  {Text: "T"},
			"rtransient": {Text: "RT"},
			"debug":      {Text: "D"},
			"tooltip":    {Text: "TT"},
			"valid":      {Text: "V"},
			"error":      {Text: "E"},
		},
	})

	require.Equal(t, &PromptResult{
		Primary:    "P",
		Right:      "R",
		Secondary:  "S",
		Transient:  "T",
		RTransient: "RT",
		Debug:      "D",
		Tooltip:    "TT",
		Valid:      "V",
		Error:      "E",
	}, result)
}

func TestExtractPromptsPartialFieldsPopulated(t *testing.T) {
	result := ExtractPrompts(&ipc.PromptResponse{
		Prompts: map[string]*ipc.Prompt{
			"primary":   {Text: "P"},
			"transient": {Text: "T"},
		},
	})

	require.Equal(t, &PromptResult{
		Primary:   "P",
		Transient: "T",
	}, result)
}

func TestExtractPromptsIgnoresUnknownKeys(t *testing.T) {
	result := ExtractPrompts(&ipc.PromptResponse{
		Prompts: map[string]*ipc.Prompt{
			"primary":     {Text: "P"},
			"unknown_key": {Text: "ignored"},
		},
	})

	require.Equal(t, &PromptResult{Primary: "P"}, result)
}
