package commands

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/wahyurudiyan/sunny-go/internal/banner"
)

// Version is sgo's own release version (not to be confused with the
// OpenAPI document version a generated project can select) — see
// CHANGELOG.md. Bump this alongside a new CHANGELOG.md entry.
const Version = "0.0.1"

var rootCmd = &cobra.Command{
	Use:     "sgo",
	Short:   "sgo CLI for Go web application boilerplating.",
	Version: Version,
	Long: color.HiWhiteString(banner.SunnyGoBannerBold) + `
sgo is an open-source CLI tool to bootstrap and evolve a Go service from a
proto contract, serving both HTTP and gRPC from a hexagonal core.

Run 'sgo init' to scaffold a new project, then 'sgo generate proto' and
'sgo generate code' to add services to it. See ARCHITECTURE.md and
PLAN.md in the repository root for the design and what's implemented.

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
