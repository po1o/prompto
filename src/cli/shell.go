package cli

import (
	"fmt"
	"os"

	"github.com/po1o/prompto/src/runtime"
	"github.com/spf13/cobra"
)

// shellCmd represents the shell command
var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Get the shell name",
	Long: `Get the shell name.

This command retrieves the name of the current shell being used.`,
	Example: `  prompto shell`,
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		flags := &runtime.Flags{
			Shell: os.Getenv("PROMPTO_SHELL"),
		}

		env := &runtime.Terminal{}
		env.Init(flags)

		fmt.Print(env.Shell())
	},
}

func init() {
	RootCmd.AddCommand(shellCmd)
}
