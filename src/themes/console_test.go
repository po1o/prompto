package themes

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/po1o/prompto/src/prompt"
	"github.com/po1o/prompto/src/runtime"
	"github.com/po1o/prompto/src/shell"
	"github.com/stretchr/testify/require"
)

// truecolor is what a console cannot show: 38;2;r;g;b or 48;2;r;g;b.
var truecolor = regexp.MustCompile(`\x1b\[[34]8;2;`)

func TestConsoleVariantsRenderConsoleSafeOutput(t *testing.T) {
	dir := t.TempDir()
	pwd := filepath.Join(dir, "work")
	require.NoError(t, os.MkdirAll(pwd, 0o755))

	for _, name := range Names() {
		content, ok := GetConsole(name)
		if !ok {
			continue
		}

		cfgPath := filepath.Join(dir, name+".console.yaml")
		require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o644))

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s panicked: %v", name, r)
				}
			}()

			e := prompt.New(&runtime.Flags{
				ConfigPath: cfgPath, Shell: shell.GENERIC, PWD: pwd, TerminalWidth: 120,
			})

			for kind, out := range map[string]string{
				"primary":    e.Primary(),
				"rprompt":    e.RPrompt(),
				"transient":  e.ExtraPrompt(prompt.Transient),
				"rtransient": e.TransientRPrompt(),
			} {
				require.NotRegexp(t, truecolor, out, "%s %s emits truecolor", name, kind)

				for _, r := range out {
					// Escape sequences and the OSC title are ASCII; so must the text be.
					require.Less(t, r, rune(0x80), "%s %s renders %U", name, kind, r)
				}
			}
		}()
	}
}
