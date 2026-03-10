package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/log"
	"github.com/po1o/prompto/src/runtime"
	"github.com/po1o/prompto/src/runtime/path"
	"github.com/po1o/prompto/src/shell"
	"github.com/po1o/prompto/src/template"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	printOutput bool
	strict      bool
	debug       bool

	initCmd = createInitCmd()
)

func init() {
	RootCmd.AddCommand(initCmd)
}

func createInitCmd() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init [shell]",
		Short: "Initialize your shell and config",
		Long: `Initialize your shell and config.

When no shell is provided, prompto tries to detect the current shell automatically.

See the documentation to initialize your shell: https://prompto.dev/docs/installation/prompt.`,
		ValidArgs: supportedShells,
		Args:      NoArgsOrOneValidArg,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 1 {
				runInit(args[0])
				return
			}

			runInit("")
		},
	}

	initCmd.Flags().BoolVarP(&printOutput, "print", "p", false, "print the init script")
	initCmd.Flags().BoolVarP(&strict, "strict", "s", false, "run in strict mode")
	initCmd.Flags().BoolVar(&debug, "debug", false, "enable/disable debug mode")
	initCmd.Flags().BoolVar(&eval, "eval", false, "output the full init script for eval")

	_ = initCmd.MarkPersistentFlagRequired("config")

	return initCmd
}

func runInit(sh string) {
	if os.Getenv("CURSOR_AGENT") == "1" {
		log.Errorf("prompto init is disabled when running inside Cursor agent mode")
		return
	}

	if debug {
		log.Enable(plain)
	}

	normalizedShell, err := resolveInitShell(sh)
	if err != nil {
		log.Error(err)
		exitcode = 1
		return
	}
	sh = normalizedShell

	if configFlag == "" {
		configFlag = config.DefaultPath()
	}

	cfg := config.Load(configFlag)
	initCache(sh)

	flags := &runtime.Flags{
		Shell:      sh,
		ConfigPath: cfg.Source,
		ConfigHash: cfg.Hash(),
		Strict:     strict,
		Debug:      debug,
		Init:       true,
		Eval:       eval,
		Plain:      plain,
		Daemon:     true,
	}

	env := &runtime.Terminal{}
	env.Init(flags)

	template.Init(env, cfg.Var, cfg.Maps)

	defer func() {
		template.SaveCache()
		if err := cache.Clear(false, shell.InitScriptName(env.Flags())); err != nil {
			log.Error(err)
		}
		cache.Close()
	}()

	feats := cfg.Features(env, true)

	var output string

	switch {
	case debug:
		output = shell.Debug(env, feats, &startTime)
	case printOutput:
		output = shell.Script(env, feats)
	default:
		output = shell.Init(env, feats)
	}

	if silent {
		return
	}

	fmt.Print(output)
}

func getFullCommand(cmd *cobra.Command, args []string) string {
	// Start with the command path
	cmdPath := cmd.CommandPath()

	// Add arguments
	if len(args) > 0 {
		cmdPath += " " + strings.Join(args, " ")
	}

	// Add flags that were actually set
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if !flag.Changed {
			return
		}

		if flag.Value.Type() == "bool" && flag.Value.String() == "true" {
			cmdPath += fmt.Sprintf(" --%s", flag.Name)
			return
		}

		if flag.Name == "config" {
			configPath := filepath.Clean(flag.Value.String())
			configPath = strings.ReplaceAll(configPath, path.Home(), "~")
			cmdPath += fmt.Sprintf(" --%s=%s", flag.Name, configPath)
			return
		}

		cmdPath += fmt.Sprintf(" --%s=%s", flag.Name, flag.Value.String())
	})

	return cmdPath
}

func initCache(sh string) {
	cache.Init(sh, cache.NoSession)
}
