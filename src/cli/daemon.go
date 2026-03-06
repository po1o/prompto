package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/po1o/prompto/src/daemon"
	"github.com/po1o/prompto/src/log"
	"github.com/po1o/prompto/src/runtime/path"

	"github.com/spf13/cobra"
)

var (
	foreground bool

	daemonCmd = &cobra.Command{
		Use:   "daemon [start|stop|restart|status|serve|log]",
		Short: "Manage the prompto daemon",
		Long: `Manage the prompto daemon for faster prompt rendering.

The daemon runs in the background and renders prompt segments asynchronously.
It automatically shuts down after being idle (no connections) for 5 minutes.

  - start:   Start the daemon (detached)
  - stop:    Stop the daemon
  - restart: Stop and start the daemon
  - status:  Check if the daemon is running
  - serve:   Run the daemon server (foreground, silent by default)
  - log:     Enable/disable daemon logging (log <path> to enable, log off to disable)`,
		ValidArgs: []string{
			"start",
			"stop",
			"restart",
			"status",
			"serve",
			"log",
		},
		Args: cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				_ = cmd.Help()
				return
			}

			switch args[0] {
			case "start":
				startDaemon()
			case "stop":
				stopDaemon()
			case "restart":
				restartDaemon()
			case "status":
				daemonStatus()
			case "serve":
				silent = true
				runDaemonServe()
			case "log":
				daemonLog(args[1:])
			default:
				_ = cmd.Help()
			}
		},
	}
)

func init() {
	daemonCmd.Flags().BoolVar(&foreground, "foreground", false, "run daemon in foreground (for debugging)")
	RootCmd.AddCommand(daemonCmd)
}

func startDaemon() {
	// Check if already running
	if daemon.IsRunning() {
		fmt.Println("daemon is already running")
		return
	}

	if foreground {
		// Enable logging to stderr for debugging
		log.Enable(false)
		runDaemonServe()
		return
	}

	if err := startDetachedDaemon(); err != nil {
		log.Error(err)
		fmt.Fprintln(os.Stderr, "failed to start daemon:", err)
		exitcode = 1
	}
}

func runDaemonServe() {
	if configFlag != "" {
		configFlag = path.ReplaceTildePrefixWithHomeDir(configFlag)
		if abs, err := filepath.Abs(configFlag); err == nil {
			configFlag = abs
		}
	}

	d, err := daemon.NewServer(configFlag)
	if err != nil {
		log.Error(err)
		fmt.Fprintln(os.Stderr, "failed to start daemon:", err)
		exitcode = 1
		return
	}

	log.Debug("daemon started")

	if err := d.Start(); err != nil {
		log.Error(err)
		fmt.Fprintln(os.Stderr, "daemon error:", err)
		exitcode = 1
	}
}

func daemonLog(args []string) {
	if len(args) == 0 {
		fmt.Println("usage: daemon log <file_path>  (enable logging)")
		fmt.Println("       daemon log off          (disable logging)")
		return
	}

	client, err := daemon.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "daemon is not running")
		exitcode = 1
		return
	}
	defer client.Close()

	ctx := context.Background()
	logPath := args[0]

	if logPath == "off" {
		logPath = ""
	}

	if err := client.SetLogging(ctx, logPath); err != nil {
		fmt.Fprintln(os.Stderr, "failed to set logging:", err)
		exitcode = 1
		return
	}

	if logPath == "" {
		fmt.Println("daemon logging disabled")
	} else {
		fmt.Println("daemon logging to", logPath)
	}
}

func stopDaemon() {
	if !daemon.IsRunning() {
		fmt.Println("daemon is not running")
		return
	}

	if err := daemon.KillDaemon(); err != nil {
		fmt.Fprintln(os.Stderr, "failed to stop daemon:", err)
		exitcode = 1
		return
	}

	fmt.Println("daemon stopped")
}

func restartDaemon() {
	if daemon.IsRunning() {
		// Try to read config from existing daemon
		_, configPath, err := daemon.GetRunningDaemonInfo()
		if err == nil && configPath != "" && configFlag == "" {
			configFlag = configPath
		}

		if err := daemon.KillDaemon(); err != nil {
			fmt.Fprintln(os.Stderr, "failed to stop daemon:", err)
			exitcode = 1
			return
		}
	}

	if err := startDetachedDaemon(); err != nil {
		log.Error(err)
		fmt.Fprintln(os.Stderr, "failed to start daemon:", err)
		exitcode = 1
	}
}

func daemonStatus() {
	if daemon.IsRunning() {
		fmt.Println("daemon is running")
	} else {
		fmt.Println("daemon is not running")
	}
}
