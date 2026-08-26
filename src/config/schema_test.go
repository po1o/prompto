package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The schema is what editors validate a user's config against. It drifted far
// enough from the parser to reject every theme in the repo, so pin the two
// together on the surface most likely to move: the top-level keys.
func TestSchemaTopLevelKeysMatchTheParser(t *testing.T) {
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}

	data, err := os.ReadFile(filepath.Join("..", "..", "themes", "schema.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &schema))

	for key := range knownLayoutTopLevelKeys {
		require.Contains(t, schema.Properties, key, "schema is missing top-level key %q", key)
	}

	for key := range schema.Properties {
		if _, removed := removedLayoutTopLevelKeys[key]; removed {
			continue
		}

		require.True(t, knownLayoutTopLevelKeys[key], "schema declares %q, which the parser rejects", key)
	}
}

// A key the parser rejects by name must be rejected by the schema too, or an
// editor gives a user porting an old config a clean bill of health right up
// until it fails to load.
func TestSchemaRejectsRemovedTopLevelKeys(t *testing.T) {
	var schema struct {
		Properties map[string]map[string]any `json:"properties"`
	}

	data, err := os.ReadFile(filepath.Join("..", "..", "themes", "schema.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &schema))

	for key := range removedLayoutTopLevelKeys {
		property, declared := schema.Properties[key]
		require.True(t, declared, "schema does not mention removed top-level key %q", key)
		require.Contains(t, property, "not", "schema does not forbid removed top-level key %q", key)
	}
}

// A removed key must be advertised as invalid, not quietly accepted.
func TestSchemaRejectsRemovedSegmentKeys(t *testing.T) {
	var schema struct {
		Definitions map[string]struct {
			Properties map[string]map[string]any `json:"properties"`
		} `json:"definitions"`
	}

	data, err := os.ReadFile(filepath.Join("..", "..", "themes", "schema.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &schema))

	for _, definition := range []string{"segment", "prompt_line"} {
		table, ok := schema.Definitions[definition]
		require.True(t, ok, definition)

		for key := range removedSeparatorKeys {
			property, declared := table.Properties[key]
			require.True(t, declared, "%s does not mention removed key %q", definition, key)
			require.Contains(t, property, "not", "%s does not forbid removed key %q", definition, key)
		}
	}
}

// The schema is closed, so a field the struct has and the schema lacks turns a
// valid config into an "additional properties are not allowed" error. Pin every
// field of both decoded types.
func TestSchemaCoversEveryStructField(t *testing.T) {
	var schema struct {
		Definitions map[string]struct {
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"definitions"`
	}

	data, err := os.ReadFile(filepath.Join("..", "..", "themes", "schema.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &schema))

	for definition, structType := range map[string]reflect.Type{
		"segment":     reflect.TypeFor[Segment](),
		"prompt_line": reflect.TypeFor[PromptLayout](),
	} {
		properties := schema.Definitions[definition].Properties
		require.NotEmpty(t, properties, definition)

		for field := range structType.Fields() {
			key, _, _ := strings.Cut(field.Tag.Get("yaml"), ",")
			if key == "" || key == "-" {
				continue
			}

			require.Contains(t, properties, key, "%s is missing %q", definition, key)
		}
	}
}
