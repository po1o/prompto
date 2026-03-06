package cli

import (
	"fmt"
	"os"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/runtime"
	"github.com/spf13/cobra"
)

// noticeCmd represents the notice command
var noticeCmd = &cobra.Command{
	Use:   "notice",
	Short: "Print the upgrade notice when a new version is available.",
	Long:  "Print the upgrade notice when a new version is available.",
	Args:  cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		env := &runtime.Terminal{}
		env.Init(&runtime.Flags{})

		cache.Init(os.Getenv("PROMPTO_SHELL"), cache.Persist)

		defer func() {
			cache.Close()
		}()

		cfg := config.Get(configFlag, false)

		if notice, hasNotice := cfg.Upgrade.Notice(); hasNotice {
			fmt.Println(notice)
		}
	},
}

func init() {
	RootCmd.AddCommand(noticeCmd)
}
