package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/po1o/prompto/src/runtime"
	"github.com/po1o/prompto/src/themes"
	"github.com/spf13/cobra"
)

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List bundled themes",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		names := themes.Names()
		output := formatColumns(names, currentTerminalWidth())
		fmt.Fprint(cmd.OutOrStdout(), output)
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <theme>",
	Short: "Write a bundled theme to the default config path",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := writeBundledTheme(cmd, args[0])
		if err != nil {
			printErrorAndExit(err)
		}
	},
}

func init() {
	configCmd.AddCommand(configListCmd, configSetCmd)
}

func writeBundledTheme(cmd *cobra.Command, name string) error {
	content, ok := themes.Get(name)
	if !ok {
		return fmt.Errorf("unknown theme %q; use `prompto config list`", name)
	}

	targetPath := resolveDefaultConfigPath()
	err := os.MkdirAll(filepath.Dir(targetPath), 0o755)
	if err != nil {
		return err
	}

	if fileExists(targetPath) && !confirmOverwrite(cmd, targetPath) {
		return fmt.Errorf("aborted")
	}

	return os.WriteFile(targetPath, []byte(content), 0o644)
}

func confirmOverwrite(cmd *cobra.Command, path string) bool {
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s already exists and will be overwritten. Continue? [y/N]: ", path)

	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr())
		return false
	}

	answer := strings.ToLower(strings.TrimSpace(line))
	_, _ = fmt.Fprintln(cmd.ErrOrStderr())
	return answer == "y" || answer == "yes"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func currentTerminalWidth() int {
	flags := &runtime.Flags{
		Shell: os.Getenv("PROMPTO_SHELL"),
	}

	env := &runtime.Terminal{}
	env.Init(flags)

	width, err := env.TerminalWidth()
	if err == nil && width > 0 {
		return width
	}

	return 80
}

func formatColumns(items []string, width int) string {
	if len(items) == 0 {
		return ""
	}

	if width <= 0 {
		width = 80
	}

	cellWidth := longestWidth(items) + 2
	columns := max(width/cellWidth, 1)

	rows := (len(items) + columns - 1) / columns
	var builder strings.Builder

	for row := range rows {
		var line strings.Builder
		for column := range columns {
			index := column*rows + row
			if index >= len(items) {
				continue
			}

			line.WriteString(items[index])

			if column == columns-1 {
				continue
			}

			nextIndex := (column+1)*rows + row
			if nextIndex >= len(items) {
				continue
			}

			padding := cellWidth - utf8.RuneCountInString(items[index])
			line.WriteString(strings.Repeat(" ", padding))
		}

		builder.WriteString(strings.TrimRight(line.String(), " "))
		builder.WriteString("\n")
	}

	return builder.String()
}

func longestWidth(items []string) int {
	longest := 0
	for _, item := range items {
		width := utf8.RuneCountInString(item)
		if width <= longest {
			continue
		}

		longest = width
	}

	return longest
}
