package commands

import (
	"github.com/spf13/cobra"
)

var helpCmd = &cobra.Command{
	Use:   "help",
	Short: "Show help for sgo CLI.",
	Long: `Show usage and help for sgo CLI.

sgo is an open-source CLI tool to bootstrap and evolve a Go service from a
proto contract, serving both HTTP and gRPC from a hexagonal core.

This build is mid-redesign; see ARCHITECTURE.md and PLAN.md in the
repository root for the target command set (sgo init, sgo generate proto,
sgo generate code, sgo ui, ...) and which phase is currently landing.

Usage:
  sgo <command> [flags]
  sgo [command] --help
`,
	Run: func(cmd *cobra.Command, args []string) {
		// Show the root command help
		rootCmd.Help()
	},
}

func init() {
	// Replaces Cobra's default help command so "help" doesn't appear
	// twice in `sgo --help`'s command list.
	rootCmd.SetHelpCommand(helpCmd)
}
