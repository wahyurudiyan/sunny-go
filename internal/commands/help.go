package commands

import (
	"github.com/spf13/cobra"
)

var helpCmd = &cobra.Command{
	Use:   "help",
	Short: "Show help for sunny CLI.",
	Long: `Show usage and help for sunny CLI.

sunny-go is an open-source CLI tool to help you bootstrap and generate Go web application boilerplate code easily.

Available commands:
  create, new    Initialize a new Go microservice project
  generate, gen, g    Generate proto contract, api, service, repository
  help               Show help for sunny CLI

Usage:
  sunny <command> [flags]
  sunny [command] --help

Examples:
  sunny create user-service
  sunny new auth-service
  sunny g proto ./proto/user.proto
  sunny help
`,
	Run: func(cmd *cobra.Command, args []string) {
		// Show the root command help
		rootCmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(helpCmd)
}
