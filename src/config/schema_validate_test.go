package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"
	yaml "go.yaml.in/yaml/v3"
)

// compileThemeSchema loads themes/schema.json into a validator.
//
// The other schema tests compare structure — which keys are declared, which are
// forbidden. That cannot catch a schema that is well-formed and still says the
// wrong thing, which is how the root ended up rejecting every theme in the repo
// once already. These tests run documents through it instead.
func compileThemeSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()

	path := filepath.Join("..", "..", "themes", "schema.json")

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var doc any
	require.NoError(t, json.Unmarshal(data, &doc))

	compiler := jsonschema.NewCompiler()
	require.NoError(t, compiler.AddResource("schema.json", doc))

	schema, err := compiler.Compile("schema.json")
	require.NoError(t, err)

	return schema
}

// readYAMLDocument decodes YAML into the any-shaped value the validator wants.
// Going through JSON normalizes the map keys: the YAML decoder produces
// map[string]any here, but the validator only accepts that exact shape.
func readYAMLDocument(t *testing.T, data []byte) any {
	t.Helper()

	var doc any
	require.NoError(t, yaml.Unmarshal(data, &doc))

	encoded, err := json.Marshal(doc)
	require.NoError(t, err)

	var normalized any
	require.NoError(t, json.Unmarshal(encoded, &normalized))

	return normalized
}

// Every bundled theme has to validate. The schema is what editors check a
// user's config against, and it once rejected all 250 files in this repo
// without anything failing.
func TestSchemaAcceptsEveryBundledTheme(t *testing.T) {
	schema := compileThemeSchema(t)

	paths, err := filepath.Glob(filepath.Join("..", "..", "themes", "*.prompto.yaml"))
	require.NoError(t, err)
	require.NotEmpty(t, paths)

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			require.NoError(t, err)

			require.NoError(t, schema.Validate(readYAMLDocument(t, data)))
		})
	}
}

// A config the parser accepts must validate, and one it rejects must not.
// Where the two disagree the schema is the only thing a user sees before their
// prompt breaks, so the pairs are asserted together.
func TestSchemaAgreesWithTheParser(t *testing.T) {
	schema := compileThemeSchema(t)

	cases := map[string]struct {
		config string
		valid  bool
	}{
		"nested instances of a segment type": {config: `
prompt:
  - segments: ["git.work"]
git:
  work:
    template: " a "
  personal:
    template: " b "
`, valid: true},
		"segment configured with options only": {config: `
prompt:
  - segments: ["git"]
git:
  options:
    fetch_status: true
`, valid: true},
		"segment type the parser knows": {config: `
prompt:
  - segments: ["v"]
v:
  type: "vim"
`, valid: true},
		"force on a segment": {config: `
prompt:
  - segments: ["t"]
t:
  type: "text"
  template: "   "
  force: true
`, valid: true},
		"segment named after its type": {config: `
prompt:
  - segments: ["git"]
git:
  template: " x "
`, valid: true},
		"segment named by the user without a type": {config: `
prompt:
  - segments: ["mysegment"]
mysegment:
  template: " x "
`, valid: false},
		"nested instances under a misspelled type": {config: `
prompt:
  - segments: ["gti.work"]
gti:
  work:
    template: " a "
`, valid: false},
		"segment type the parser rejects": {config: `
prompt:
  - segments: ["c"]
c:
  type: "claude"
`, valid: false},
		"tooltip without a type": {config: `
prompt:
  - segments: ["s"]
s:
  type: "session"
tooltips:
  - tips: ["aws"]
`, valid: false},
		"removed key on an extra prompt line": {config: `
prompt:
  - segments: ["s"]
s:
  type: "session"
debug_prompt:
  template: "x"
  powerline_symbol: ">"
`, valid: false},
		"engine glyph written by hand": {config: `
prompt:
  - segments: ["s"]
s:
  type: "session"
  leading_glyph: "<"
`, valid: false},
		"legacy blocks document": {config: `
blocks:
  - type: "prompt"
    segments: []
`, valid: false},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			document := readYAMLDocument(t, []byte(test.config))
			schemaErr := schema.Validate(document)
			_, parseErr := ParseLayoutYAML([]byte(test.config))

			if test.valid {
				require.NoError(t, schemaErr, "schema rejects a config the parser accepts")
				require.NoError(t, parseErr, "parser rejects it too — fix the case, not the schema")
				return
			}

			require.Error(t, schemaErr, "schema accepts a config the parser rejects")
			require.Error(t, parseErr, "parser accepts it — the schema is stricter than the loader")
		})
	}
}

// The schema has to stay loadable by the editors that consume it. The compiler
// resolves every $ref and fails on one that points nowhere, so compiling the
// real file is the assertion — a dangling ref is otherwise invisible until an
// editor quietly validates nothing.
func TestSchemaCompiles(t *testing.T) {
	compileThemeSchema(t)
}

// `git.` names no instance. The parser used to infer `git` from it while the
// schema's dotted pattern requires a character after the dot, which made it the
// one input the two disagreed on.
func TestSchemaAndParserRejectATrailingDotSegmentName(t *testing.T) {
	schema := compileThemeSchema(t)

	raw := `
prompt:
  - segments: ["git."]

git.:
  template: " x "
`

	require.Error(t, schema.Validate(readYAMLDocument(t, []byte(raw))))

	_, err := ParseLayoutYAML([]byte(raw))
	require.Error(t, err)
}
