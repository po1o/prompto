package cli

import (
	"fmt"
	"io"
	"os"
	"sync"

	daemonpkg "github.com/jandedobbeleer/oh-my-posh/src/daemon"
	"github.com/spf13/cobra"
)

type managedDaemon interface {
	Stop()
}

type daemonFactory func() managedDaemon

type daemonController struct {
	mu       sync.Mutex
	factory  daemonFactory
	instance managedDaemon
}

func newDaemonController(factory daemonFactory) *daemonController {
	if factory == nil {
		factory = func() managedDaemon {
			return daemonpkg.New(nil)
		}
	}

	return &daemonController{
		factory: factory,
	}
}

func (controller *daemonController) Start() bool {
	controller.mu.Lock()
	defer controller.mu.Unlock()

	if controller.instance != nil {
		return false
	}

	controller.instance = controller.factory()
	return true
}

func (controller *daemonController) Stop() bool {
	controller.mu.Lock()
	defer controller.mu.Unlock()

	if controller.instance == nil {
		return false
	}

	controller.instance.Stop()
	controller.instance = nil
	return true
}

func (controller *daemonController) Restart() {
	controller.Stop()
	controller.Start()
}

func (controller *daemonController) Running() bool {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	return controller.instance != nil
}

var (
	daemonRuntime = newDaemonController(nil)
	daemonCmd     = createDaemonCmd()
)

func init() {
	RootCmd.AddCommand(daemonCmd)
}

func createDaemonCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "daemon [start|stop|restart|status|serve]",
		Short: "Manage the oh-my-posh daemon",
		Long: `Manage the oh-my-posh daemon for faster prompt rendering.

Available commands:
  - start:   Start the daemon
  - stop:    Stop the daemon
  - restart: Restart the daemon
  - status:  Check if the daemon is running
  - serve:   Run the daemon server mode`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runDaemonAction(args[0], daemonRuntime, os.Stdout)
		},
	}
}

func runDaemonAction(action string, controller *daemonController, out io.Writer) error {
	switch action {
	case "start", "serve":
		if controller.Start() {
			fmt.Fprintln(out, "daemon started")
			return nil
		}

		fmt.Fprintln(out, "daemon is already running")
		return nil
	case "stop":
		if controller.Stop() {
			fmt.Fprintln(out, "daemon stopped")
			return nil
		}

		fmt.Fprintln(out, "daemon is not running")
		return nil
	case "restart":
		controller.Restart()
		fmt.Fprintln(out, "daemon restarted")
		return nil
	case "status":
		if controller.Running() {
			fmt.Fprintln(out, "daemon is running")
			return nil
		}

		fmt.Fprintln(out, "daemon is not running")
		return nil
	}

	return fmt.Errorf("unknown daemon action: %s", action)
}
