package shell

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNuFeatures(t *testing.T) {
	got := allFeatures.Lines(NU).String("// these are the features")

	want := `// these are the features
$env.TRANSIENT_PROMPT_COMMAND = {|| _prompto_get_prompt transient }
^$_prompto_executable upgrade --auto
^$_prompto_executable notice
enable_prompto_daemon`

	assert.Equal(t, want, got)
}

func TestQuoteNuStr(t *testing.T) {
	tests := []struct {
		str      string
		expected string
	}{
		{str: "", expected: "''"},
		{str: `/tmp/"omp's dir"/prompto`, expected: `"/tmp/\"omp's dir\"/prompto"`},
		{str: `C:/tmp\omp's dir/prompto.exe`, expected: `"C:/tmp\\omp's dir/prompto.exe"`},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.expected, quoteNuStr(tc.str), fmt.Sprintf("quoteNuStr: %s", tc.str))
	}
}
