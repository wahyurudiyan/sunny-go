package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var helpCmd = &cobra.Command{
	Use:     "help",
	Aliases: []string{"-h", "--help"},
	Short:   "Show help for sunny CLI.",
	Long: `Show usage and help for sunny CLI.

Usage:
  sunny help
  sunny -h
  sunny --help
  sunny
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("help called")
	},
}

func init() {
	rootCmd.AddCommand(helpCmd)
}
