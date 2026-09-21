package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/wahyurudiyan/sunny-go/internal/webui"
)

var uiPort int

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Start the localhost web UI",
	Long: `Start a localhost-only web UI covering the same ground as the CLI:
create a new project (the same selections as the interactive wizard),
and — inside an existing project — a dashboard of its services,
generated-vs-owned file status, and an sgo.yaml viewer/editor.

Binds to 127.0.0.1 only; there is no auth, since it never listens on
anything but loopback.

Example:
  sgo ui
  sgo ui --port 4747
`,
	Run: func(cmd *cobra.Command, args []string) {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("🌐 sgo ui running at http://127.0.0.1:%d\n", uiPort)
		if err := webui.Serve(dir, uiPort); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	uiCmd.Flags().IntVar(&uiPort, "port", 4747, "Port to bind the web UI to")
	rootCmd.AddCommand(uiCmd)
}
