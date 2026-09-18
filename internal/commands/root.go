package commands

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/wahyurudiyan/sunny-go/internal/banner"
)

var rootCmd = &cobra.Command{
	Use:   "sgo",
	Short: "sgo CLI for Go web application boilerplating.",
	Long: color.HiWhiteString(banner.SunnyGoBannerBold) + `
sgo is an open-source CLI tool to bootstrap and evolve a Go service from a
proto contract, serving both HTTP and gRPC from a hexagonal core.

This build is mid-redesign: the command set is being rebuilt per
ARCHITECTURE.md and PLAN.md in the repository root. Run 'sgo help' for
what's available right now.

Usage:
  sgo <command> [flags]
  sgo [command] --help
`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Println(err)
	}
}
