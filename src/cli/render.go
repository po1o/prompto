package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/po1o/prompto/src/daemon"
	"github.com/po1o/prompto/src/daemon/ipc"
	"github.com/po1o/prompto/src/runtime"
	"github.com/po1o/prompto/src/runtime/path"
	"github.com/po1o/prompto/src/shell"

	"github.com/spf13/cobra"
)

const (
	renderTimeout = 10 * time.Second
)

var renderOutputEscaper = strings.NewReplacer(
	`\`, `\\`,
	"\n", `\n`,
	"\r", `\r`,
)

var (
	pwd           string
	pswd          string
	status        int
	pipestatus    string
	timing        float64
	stackCount    int
	terminalWidth int
	eval          bool
	cleared       bool
	jobCount      int
	shellVersion  string
	noStatus      bool
	column        int
	escape        bool
	renderForce   bool

	pid           int
	repaint       bool
	renderVimMode string
)

var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "Render prompts via the daemon",
	Long: `Render all prompts via the daemon for faster display.

The daemon computes segments asynchronously and streams updates.
After a short timeout (100ms), partial results are returned with
cached values for slow segments. Updates stream as segments complete.

Output format (one per line):
  primary:<text>
  right:<text>
  secondary:<text>
  ...`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if shellName == "" {
			shellName = shell.GENERIC
		}

		if shellName != shell.GENERIC {
			normalizedShell, err := normalizeSupportedShell(shellName)
			if err != nil {
				exitcode = 1
				return
			}
			shellName = normalizedShell
		}

		if configFlag != "" {
			configFlag = path.ReplaceTildePrefixWithHomeDir(configFlag)
			if abs, err := filepath.Abs(configFlag); err == nil {
				configFlag = abs
			}
		}

		flags := &runtime.Flags{
			ConfigPath:    configFlag,
			PWD:           pwd,
			PSWD:          pswd,
			ErrorCode:     status,
			PipeStatus:    pipestatus,
			ExecutionTime: timing,
			StackCount:    stackCount,
			TerminalWidth: terminalWidth,
			Shell:         shellName,
			ShellVersion:  shellVersion,
			Plain:         plain,
			Cleared:       cleared,
			Eval:          eval,
			NoExitCode:    noStatus,
			Column:        column,
			JobCount:      jobCount,
			Escape:        escape,
			Force:         renderForce,
			VimMode:       renderVimMode,
		}

		if err := renderViaDaemon(flags, pid, repaint); err != nil {
			fmt.Fprintln(os.Stderr, err)
			exitcode = 1
		}
	},
}

func init() {
	renderCmd.Flags().StringVar(&pwd, "pwd", "", "current working directory")
	renderCmd.Flags().StringVar(&pswd, "pswd", "", "current working directory (according to pwsh)")
	renderCmd.Flags().StringVar(&shellName, "shell", "", "the shell to render for")
	renderCmd.Flags().StringVar(&shellVersion, "shell-version", "", "the shell version")
	renderCmd.Flags().IntVar(&status, "status", 0, "last known status code")
	renderCmd.Flags().BoolVar(&noStatus, "no-status", false, "no valid status code (cancelled or no command yet)")
	renderCmd.Flags().StringVar(&pipestatus, "pipestatus", "", "the PIPESTATUS array")
	renderCmd.Flags().Float64Var(&timing, "execution-time", 0, "timing of the last command")
	renderCmd.Flags().IntVarP(&stackCount, "stack-count", "s", 0, "number of locations on the stack")
	renderCmd.Flags().IntVarP(&terminalWidth, "terminal-width", "w", 0, "width of the terminal")
	renderCmd.Flags().BoolVar(&cleared, "cleared", false, "do we have a clear terminal or not")
	renderCmd.Flags().BoolVar(&eval, "eval", false, "output the prompt for eval")
	renderCmd.Flags().IntVar(&column, "column", 0, "the column position of the cursor")
	renderCmd.Flags().IntVar(&jobCount, "job-count", 0, "number of background jobs")
	renderCmd.Flags().BoolVar(&escape, "escape", true, "escape the ANSI sequences for the shell")
	renderCmd.Flags().BoolVarP(&renderForce, "force", "f", false, "force rendering the segments")
	renderCmd.Flags().IntVar(&pid, "pid", 0, "shell process id")
	renderCmd.Flags().BoolVar(&repaint, "repaint", false, "vim mode repaint (soft cancel, reuse computations)")
	renderCmd.Flags().StringVar(&renderVimMode, "vim-mode", "", "current vim mode (insert, normal, visual, replace)")
	RootCmd.AddCommand(renderCmd)
}

func renderViaDaemon(flags *runtime.Flags, pid int, repaint bool) error {
	silent = true
	client, err := daemon.ConnectOrStart(startDetachedDaemon)
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), renderTimeout)
	defer cancel()

	return client.RenderPrompt(ctx, flags, pid, "", nil, repaint, func(resp *ipc.PromptResponse) bool {
		outputPrompts(resp)
		return resp.Type != "complete"
	})
}

func outputPrompts(resp *ipc.PromptResponse) {
	if resp == nil || resp.Prompts == nil {
		return
	}

	// Output each prompt with a prefix for shell parsing
	// Format: type:text (text can contain newlines, shell handles it)
	//
	// IMPORTANT: Always output primary and right prompts even if empty.
	// The shell keeps previous values if a prompt type isn't sent,
	// so we must send empty values to clear stale prompts (e.g., git segment
	// persisting after leaving a repo).
	alwaysOutput := map[string]bool{"primary": true, "right": true}
	promptTypes := []string{"primary", "right", "secondary", "transient", "rtransient", "debug", "valid", "error"}

	for _, pt := range promptTypes {
		if p, ok := resp.Prompts[pt]; ok {
			// Always output primary/right; only output others if non-empty
			if alwaysOutput[pt] || p.Text != "" {
				fmt.Printf("%s:%s\n", pt, encodeRenderOutputText(p.Text))
			}
		}
	}

	// Output status line so shell knows when a batch is complete
	// "update" = more updates may come, "complete" = all segments done
	fmt.Printf("status:%s\n", resp.Type)
}

func encodeRenderOutputText(text string) string {
	if plain {
		return text
	}

	return renderOutputEscaper.Replace(text)
}
