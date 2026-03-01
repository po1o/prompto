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
	renderPwd           string
	renderPSwd          string
	renderShell         string
	renderShellVersion  string
	renderStatus        int
	renderPipeStatus    string
	renderTiming        float64
	renderStackCount    int
	renderTerminalWidth int
	renderNoStatus      bool
	renderColumn        int
	renderJobCount      int
	renderEscape        bool
	renderForce         bool
	renderSessionID     string
	renderPID           int
	renderRepaint       bool
	renderMaxUpdates    int
	renderVimMode       string
)

var renderCmd = createRenderCmd()

const renderUpdateTimeout = 100 * time.Millisecond

func init() {
	RootCmd.AddCommand(renderCmd)
}

func createRenderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render prompts via the daemon service",
		RunE: func(c *cobra.Command, _ []string) error {
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

			updateTimeout := resolveRenderUpdateTimeout()

			return renderWithDaemon(
				daemonRuntime,
				flags,
				sessionID,
				renderRepaint,
				renderMaxUpdates,
				updateTimeout,
				os.Stdout,
			)
		},
	}

	cmd.Flags().StringVar(&renderPwd, "pwd", "", "current working directory")
	cmd.Flags().StringVar(&renderPSwd, "pswd", "", "current working directory (according to pwsh)")
	cmd.Flags().StringVar(&renderShell, "shell", "", "the shell to render for")
	cmd.Flags().StringVar(&renderShellVersion, "shell-version", "", "the shell version")
	cmd.Flags().IntVar(&renderStatus, "status", 0, "last known status code")
	cmd.Flags().BoolVar(&renderNoStatus, "no-status", false, "no valid status code")
	cmd.Flags().StringVar(&renderPipeStatus, "pipestatus", "", "the PIPESTATUS array")
	cmd.Flags().Float64Var(&renderTiming, "execution-time", 0, "timing of the last command")
	cmd.Flags().IntVarP(&renderStackCount, "stack-count", "s", 0, "number of locations on the stack")
	cmd.Flags().IntVarP(&renderTerminalWidth, "terminal-width", "w", 0, "width of the terminal")
	cmd.Flags().IntVar(&renderColumn, "column", 0, "the column position of the cursor")
	cmd.Flags().IntVar(&renderJobCount, "job-count", 0, "number of background jobs")
	cmd.Flags().BoolVar(&renderEscape, "escape", true, "escape ANSI sequences for the shell")
	cmd.Flags().BoolVarP(&renderForce, "force", "f", false, "force rendering the segments")
	cmd.Flags().StringVar(&renderSessionID, "session-id", "", "session identifier")
	cmd.Flags().IntVar(&renderPID, "pid", 0, "shell process id (used as default session identifier)")
	cmd.Flags().BoolVar(&renderRepaint, "repaint", false, "render as repaint request")
	cmd.Flags().IntVar(&renderMaxUpdates, "max-updates", 10, "maximum streamed updates to print")
	cmd.Flags().StringVar(&renderVimMode, "vim-mode", "", "current vim mode (insert, normal, visual, replace)")

	return cmd
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

func resolveRenderUpdateTimeout() time.Duration {
	return renderUpdateTimeout
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
