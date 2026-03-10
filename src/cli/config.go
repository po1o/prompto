package cli

import (
	"os"
	"path/filepath"

	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/daemon"
	"github.com/po1o/prompto/src/runtime/path"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Interact with the config",
	Long: `Interact with the config.

You can edit the active config, list bundled themes, set a bundled theme,
or render a config preview image.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		_ = cmd.Help()
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open the active config in $EDITOR",
	Args:  cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		configPath := resolveConfigPath()
		if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
			printErrorAndExit(err)
			return
		}

		exitcode = editFileWithEditor(configPath)
	},
}

func resolveConfigPath() string {
	if configFlag == "" {
		pid, daemonConfigPath, err := daemon.GetRunningDaemonInfo()
		if err == nil && pid > 0 && daemon.IsProcessRunning(pid) && daemonConfigPath != "" {
			configFlag = daemonConfigPath
		}
	}

	if configFlag == "" {
		configFlag = config.DefaultPath()
	}

	configFlag = path.ReplaceTildePrefixWithHomeDir(configFlag)

	if absPath, err := filepath.Abs(configFlag); err == nil {
		configFlag = absPath
	}

	return filepath.Clean(configFlag)
}

func resolveDefaultConfigPath() string {
	defaultPath := path.ReplaceTildePrefixWithHomeDir(config.DefaultPath())

	if absPath, err := filepath.Abs(defaultPath); err == nil {
		defaultPath = absPath
	}

	return filepath.Clean(defaultPath)
}

func init() {
	configCmd.AddCommand(configEditCmd)
	RootCmd.AddCommand(configCmd)
}

func printErrorAndExit(err error) {
	if err == nil {
		return
	}

	os.Stdout.WriteString(err.Error() + "\n")
	exitcode = 1
}
