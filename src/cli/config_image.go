package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/cli/image"
	"github.com/po1o/prompto/src/config"
	"github.com/po1o/prompto/src/prompt"
	"github.com/po1o/prompto/src/runtime"
	"github.com/po1o/prompto/src/runtime/path"
	"github.com/po1o/prompto/src/shell"
	"github.com/po1o/prompto/src/template"
	"github.com/po1o/prompto/src/terminal"

	"github.com/spf13/cobra"
)

var (
	author            string
	colorSettingsFile string
	bgColor           string
	outputImage       string
)

// imageCmd represents the image command
var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Export your config to an image",
	Long: `Export your config to an image.

You can tweak the output by using additional flags:

- cursor-padding: the padding of the prompt cursor
- rprompt-offset: the offset of the right prompt
- settings: JSON file with overrides

Example usage:

> prompto config image --config ~/.config/prompto/config.yaml

Exports the config to an image file called myconfig.png in the current working directory.

> prompto config image --config ~/.config/prompto/config.yaml --output ~/mytheme.png

Exports the config to an image file ~/mytheme.png.

> prompto config image --config ~/.config/prompto/config.yaml --settings ~/.image.settings.json

Exports the config to an image file using customized output settings.`,
	Args: cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		cache.Init(os.Getenv("PROMPTO_SHELL"))

		setConfigFlag()

		cfg := config.Load(configFlag)

		flags := &runtime.Flags{
			ConfigPath:    cfg.Source,
			Shell:         shell.GENERIC,
			TerminalWidth: 120,
		}

		env := &runtime.Terminal{}
		env.Init(flags)

		template.Init(env, cfg.Var, cfg.Maps)

		defer func() {
			template.SaveCache()
			cache.Close()
		}()

		// set sane defaults for things we don't print
		cfg.ConsoleTitleTemplate = ""
		cfg.PWD = ""
		cfg.ShellIntegration = false

		terminal.Init(shell.GENERIC)
		terminal.BackgroundColor = cfg.TerminalBackground.ResolveTemplate()
		terminal.Colors = cfg.MakeColors(env)

		eng := &prompt.Engine{
			Config:       cfg,
			Env:          env,
			LayoutConfig: cfg.Layout,
		}

		settings, err := image.LoadSettings(colorSettingsFile)
		if err != nil {
			settings = &image.Settings{
				Colors:          image.NewColors(),
				Author:          author,
				BackgroundColor: bgColor,
			}
		}

		if settings.Colors == nil {
			settings.Colors = image.NewColors()
		}

		if settings.Cursor == "" {
			settings.Cursor = "_"
		}

		primaryPrompt := eng.Primary()

		imageCreator := &image.Renderer{
			AnsiString: primaryPrompt,
			Settings:   *settings,
		}

		if outputImage != "" {
			imageCreator.Path = cleanOutputPath(outputImage)
		}

		err = imageCreator.Init(env)
		if err != nil {
			fmt.Print(err.Error())
			return
		}

		err = imageCreator.SavePNG()
		if err != nil {
			fmt.Print(err.Error())
		}
	},
}

func init() {
	imageCmd.Flags().StringVar(&author, "author", "", "config author")
	imageCmd.Flags().StringVar(&bgColor, "background-color", "", "image background color")
	imageCmd.Flags().StringVarP(&outputImage, "output", "o", "", "image file (.png) to export to")
	imageCmd.Flags().StringVar(&colorSettingsFile, "settings", "", "color settings file to override ANSI color codes and metadata")

	// deprecated flags
	_ = imageCmd.Flags().MarkHidden("author")
	_ = imageCmd.Flags().MarkHidden("background-color")

	configCmd.AddCommand(imageCmd)
}

func setConfigFlag() {
	configFlag = resolveConfigPath()
}

func cleanOutputPath(output string) string {
	output = path.ReplaceTildePrefixWithHomeDir(output)

	if !filepath.IsAbs(output) {
		if absPath, err := filepath.Abs(output); err == nil {
			output = absPath
		}
	}

	return filepath.Clean(output)
}
