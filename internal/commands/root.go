package commands

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/wahyurudiyan/sunny-go/internal/banner"
)

var rootCmd = &cobra.Command{
	Use:   "sunny",
	Short: "Sunny CLI for Go web application boilerplating.",
	Long: color.HiWhiteString(banner.SunnyGoBannerBold) + `
SunnyGo is an open-source CLI tool to help you bootstrap and generate Go web application boilerplate code easily.

Available commands:
  create         Initialize a new project
  generate|gen|g Generate proto contract, api, service, repository
  help           Show help for sunny CLI

Usage:
  sunny <command> [flags]
  sunny [command] --help

Examples:
  sunny create my-api
  sunny g api user
  sunny help
`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Println(err)
	}
}
