package themes

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/po1o/prompto/src/prompt"
	"github.com/po1o/prompto/src/runtime"
	"github.com/po1o/prompto/src/shell"
	"github.com/stretchr/testify/require"
)

// TestGolden renders every bundled theme and writes the result to the path in
// GOLDEN. It exists to be run twice — once on a known-good tree, once on a
// change — and the two files compared, which is the only practical way to tell
// a deliberate rendering change from an accidental one across 125 themes.
//
// It is skipped by default: it writes a file, and the comparison is the point,
// not the rendering.
func TestGolden(t *testing.T) {
	out := os.Getenv("GOLDEN")
	if out == "" {
		t.Skip("set GOLDEN=<path> to capture a rendering baseline")
	}

	// The rendered prompt contains the working directory, so the two runs being
	// compared have to share one. It is derived from the output path rather
	// than being a second knob: both runs already point GOLDEN at the same
	// directory, which is what makes them comparable in the first place.
	dir := t.TempDir()
	pwd := filepath.Join(filepath.Dir(out), "golden-work")
	require.NoError(t, os.MkdirAll(pwd, 0o755))

	var report strings.Builder

	names := Names()
	sort.Strings(names)

	for _, name := range names {
		content, ok := Get(name)
		if !ok {
			continue
		}

		configPath := filepath.Join(dir, name+".yaml")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

		func() {
			// A theme that panics is a result worth recording rather than a
			// reason to abandon the other 124.
			defer func() {
				if recovered := recover(); recovered != nil {
					fmt.Fprintf(&report, "%s\tPANIC\t%v\n", name, recovered)
				}
			}()

			engine := prompt.New(&runtime.Flags{
				ConfigPath:    configPath,
				Shell:         shell.GENERIC,
				PWD:           pwd,
				TerminalWidth: 120,
			})

			fmt.Fprintf(&report, "%s\tprimary\t%q\n", name, engine.Primary())
			fmt.Fprintf(&report, "%s\trprompt\t%q\n", name, engine.RPrompt())
			fmt.Fprintf(&report, "%s\ttransient\t%q\n", name, engine.ExtraPrompt(prompt.Transient))
			fmt.Fprintf(&report, "%s\trtransient\t%q\n", name, engine.TransientRPrompt())
		}()
	}

	require.NoError(t, os.WriteFile(out, []byte(report.String()), 0o644))
}
