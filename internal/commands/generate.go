package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"gen", "g"},
	Short:   "Generate proto contract, api, service, or repository.",
	Long: `Generate code for proto contract, api, service, or repository.

Usage:
  sunny generate proto <path_to_directory/filename.go>
  sunny gen api <path_to_directory/filename.go>
  sunny g service <path_to_directory/filename.go>
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("generate called")
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
