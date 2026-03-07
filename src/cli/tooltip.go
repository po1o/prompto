package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/po1o/prompto/src/daemon"
	"github.com/po1o/prompto/src/prompt"
	"github.com/po1o/prompto/src/runtime"
	"github.com/po1o/prompto/src/runtime/path"
	"github.com/po1o/prompto/src/shell"
	"github.com/spf13/cobra"
)

var tooltipCommand string

var tooltipCmd = &cobra.Command{
	Use:   "tooltip",
	Short: "Render tooltip only",
	Args:  cobra.NoArgs,
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
			Type:          prompt.TOOLTIP,
			Command:       tooltipCommand,
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
			NoExitCode:    noStatus,
			Column:        column,
			JobCount:      jobCount,
			Escape:        escape,
		}

		silent = true
		client, err := daemon.ConnectOrStart(startDetachedDaemon)
		if err != nil {
			exitcode = 1
			return
		}
		defer client.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resp, err := client.RenderPromptSync(ctx, flags, pid, "", nil, false)
		if err != nil {
			exitcode = 1
			return
		}

		result := daemon.ExtractPrompts(resp)
		fmt.Print(result.Tooltip)
	},
}

func init() {
	tooltipCmd.Flags().StringVar(&pwd, "pwd", "", "current working directory")
	tooltipCmd.Flags().StringVar(&pswd, "pswd", "", "current working directory (according to pwsh)")
	tooltipCmd.Flags().StringVar(&shellName, "shell", "", "the shell to render for")
	tooltipCmd.Flags().StringVar(&shellVersion, "shell-version", "", "the shell version")
	tooltipCmd.Flags().IntVar(&status, "status", 0, "last known status code")
	tooltipCmd.Flags().BoolVar(&noStatus, "no-status", false, "no valid status code (cancelled or no command yet)")
	tooltipCmd.Flags().StringVar(&pipestatus, "pipestatus", "", "the PIPESTATUS array")
	tooltipCmd.Flags().Float64Var(&timing, "execution-time", 0, "timing of the last command")
	tooltipCmd.Flags().IntVarP(&stackCount, "stack-count", "s", 0, "number of locations on the stack")
	tooltipCmd.Flags().IntVarP(&terminalWidth, "terminal-width", "w", 0, "width of the terminal")
	tooltipCmd.Flags().BoolVar(&cleared, "cleared", false, "do we have a clear terminal or not")
	tooltipCmd.Flags().IntVar(&column, "column", 0, "the column position of the cursor")
	tooltipCmd.Flags().IntVar(&jobCount, "job-count", 0, "number of background jobs")
	tooltipCmd.Flags().StringVar(&tooltipCommand, "command", "", "tooltip command")
	tooltipCmd.Flags().BoolVar(&escape, "escape", true, "escape the ANSI sequences for the shell")
	tooltipCmd.Flags().IntVar(&pid, "pid", 0, "shell process id")
	RootCmd.AddCommand(tooltipCmd)
}
