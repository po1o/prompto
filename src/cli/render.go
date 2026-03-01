package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	daemonpkg "github.com/jandedobbeleer/oh-my-posh/src/daemon"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime"
	"github.com/jandedobbeleer/oh-my-posh/src/shell"
	"github.com/spf13/cobra"
)

var (
	renderPwd             string
	renderPSwd            string
	renderShell           string
	renderShellVersion    string
	renderStatus          int
	renderPipeStatus      string
	renderTiming          float64
	renderStackCount      int
	renderTerminalWidth   int
	renderNoStatus        bool
	renderColumn          int
	renderJobCount        int
	renderEscape          bool
	renderForce           bool
	renderSessionID       string
	renderPID             int
	renderRepaint         bool
	renderMaxUpdates      int
	renderUpdateTimeoutMS int
	renderVimMode         string
)

var renderCmd = createRenderCmd()

func init() {
	RootCmd.AddCommand(renderCmd)
}

func createRenderCmd() *cobra.Command {
	renderCmd := &cobra.Command{
		Use:   "render",
		Short: "Render prompts via the daemon service",
		RunE: func(_ *cobra.Command, _ []string) error {
			sh := renderShell
			if sh == "" {
				sh = shell.GENERIC
			}

			sessionID := resolveRenderSessionID(renderSessionID, renderPID)

			flags := &runtime.Flags{
				ConfigPath:    configFlag,
				PWD:           renderPwd,
				PSWD:          renderPSwd,
				ErrorCode:     renderStatus,
				PipeStatus:    renderPipeStatus,
				ExecutionTime: renderTiming,
				StackCount:    renderStackCount,
				TerminalWidth: renderTerminalWidth,
				Shell:         sh,
				ShellVersion:  renderShellVersion,
				Plain:         plain,
				NoExitCode:    renderNoStatus,
				Column:        renderColumn,
				JobCount:      renderJobCount,
				Escape:        renderEscape,
				Force:         renderForce,
				VimMode:       renderVimMode,
			}

			return renderWithDaemon(
				daemonRuntime,
				flags,
				sessionID,
				renderRepaint,
				renderMaxUpdates,
				time.Duration(renderUpdateTimeoutMS)*time.Millisecond,
				os.Stdout,
			)
		},
	}

	renderCmd.Flags().StringVar(&renderPwd, "pwd", "", "current working directory")
	renderCmd.Flags().StringVar(&renderPSwd, "pswd", "", "current working directory (according to pwsh)")
	renderCmd.Flags().StringVar(&renderShell, "shell", "", "the shell to render for")
	renderCmd.Flags().StringVar(&renderShellVersion, "shell-version", "", "the shell version")
	renderCmd.Flags().IntVar(&renderStatus, "status", 0, "last known status code")
	renderCmd.Flags().BoolVar(&renderNoStatus, "no-status", false, "no valid status code")
	renderCmd.Flags().StringVar(&renderPipeStatus, "pipestatus", "", "the PIPESTATUS array")
	renderCmd.Flags().Float64Var(&renderTiming, "execution-time", 0, "timing of the last command")
	renderCmd.Flags().IntVarP(&renderStackCount, "stack-count", "s", 0, "number of locations on the stack")
	renderCmd.Flags().IntVarP(&renderTerminalWidth, "terminal-width", "w", 0, "width of the terminal")
	renderCmd.Flags().IntVar(&renderColumn, "column", 0, "the column position of the cursor")
	renderCmd.Flags().IntVar(&renderJobCount, "job-count", 0, "number of background jobs")
	renderCmd.Flags().BoolVar(&renderEscape, "escape", true, "escape ANSI sequences for the shell")
	renderCmd.Flags().BoolVarP(&renderForce, "force", "f", false, "force rendering the segments")
	renderCmd.Flags().StringVar(&renderSessionID, "session-id", "", "session identifier")
	renderCmd.Flags().IntVar(&renderPID, "pid", 0, "shell process id (used as default session identifier)")
	renderCmd.Flags().BoolVar(&renderRepaint, "repaint", false, "render as repaint request")
	renderCmd.Flags().IntVar(&renderMaxUpdates, "max-updates", 10, "maximum streamed updates to print")
	renderCmd.Flags().IntVar(&renderUpdateTimeoutMS, "update-timeout", 75, "update wait timeout in milliseconds")
	renderCmd.Flags().StringVar(&renderVimMode, "vim-mode", "", "current vim mode (insert, normal, visual, replace)")

	return renderCmd
}

func resolveRenderSessionID(explicitSessionID string, pid int) string {
	if explicitSessionID != "" {
		return explicitSessionID
	}

	if pid > 0 {
		return strconv.Itoa(pid)
	}

	sessionID := os.Getenv("POSH_SESSION_ID")
	if sessionID != "" {
		return sessionID
	}

	return "default"
}

func renderWithDaemon(
	controller *daemonController,
	flags *runtime.Flags,
	sessionID string,
	repaint bool,
	maxUpdates int,
	updateTimeout time.Duration,
	out io.Writer,
) error {
	instance := controller.EnsureStarted()
	response := instance.StartRender(daemonpkg.RenderRequest{
		SessionID: sessionID,
		Flags:     flags,
		Repaint:   repaint,
	})

	if response.Type == "stopped" {
		return fmt.Errorf("daemon is stopped")
	}

	writePromptBundle(out, response.Bundle)
	sequence := response.Sequence

	for range maxUpdates {
		ctx, cancel := context.WithTimeout(context.Background(), updateTimeout)
		update, ok := instance.NextUpdate(ctx, sessionID, sequence)
		cancel()
		if !ok {
			break
		}

		sequence = update.Sequence
		writePromptBundle(out, update.Bundle)
		fmt.Fprintln(out, "status:update")
	}

	fmt.Fprintln(out, "status:complete")
	return nil
}

func writePromptBundle(out io.Writer, bundle daemonpkg.PromptBundle) {
	fmt.Fprintf(out, "primary:%s\n", bundle.Primary)
	fmt.Fprintf(out, "right:%s\n", bundle.RPrompt)

	if bundle.Secondary != "" {
		fmt.Fprintf(out, "secondary:%s\n", bundle.Secondary)
	}

	if bundle.Transient != "" {
		fmt.Fprintf(out, "transient:%s\n", bundle.Transient)
	}
}
