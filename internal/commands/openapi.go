package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/openapigen"
	"github.com/wahyurudiyan/sunny-go/internal/openapiui"
)

var openapiCmd = &cobra.Command{
	Use:   "openapi",
	Short: "Work with OpenAPI documents",
}

var openapiValidateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Validate an OpenAPI document against the real OpenAPI meta-schema",
	Long: `Validate an OpenAPI document — json or yaml, 3.0.x or 3.1.x (auto-detected
from its own "openapi" field) — against the official OpenAPI JSON Schema
meta-schema. This checks the document's own well-formedness, not
whether it matches any particular project's generated routes.

With no path, validates the current project's own generated
docs/openapi.<ext> (run 'sgo generate openapi' first if it doesn't exist
yet). With a path, validates that file instead — sgo-generated or
hand-authored, and doesn't need to be run from inside an sgo project.

Examples:
  sgo openapi validate
  sgo openapi validate docs/openapi.yaml
  sgo openapi validate ./some-other-api.json
`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path, err := resolveValidatePath(args)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("❌ Error: failed to read %s: %v\n", path, err)
			os.Exit(1)
		}

		if err := openapigen.Validate(data); err != nil {
			fmt.Printf("❌ %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ %s is a valid OpenAPI document\n", path)
	},
}

// resolveValidatePath returns the explicit path argument if given,
// otherwise the current project's own generated doc path — which needs
// sgo.yaml loaded just to know its extension (yaml vs json), not to
// validate the config itself.
func resolveValidatePath(args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}

	cfg, err := loadProjectConfig()
	if err != nil {
		return "", fmt.Errorf("no path given and %w", err)
	}

	return openapigen.Path(".", cfg.OpenAPI.Format), nil
}

var openapiUIPort int

var openapiUICmd = &cobra.Command{
	Use:   "ui",
	Short: "Browse the project's generated OpenAPI document in a live viewer",
	Long: `Serve the current project's already-generated docs/openapi.<ext>
through an embedded, offline Redoc viewer at http://127.0.0.1:<port>.

Binds to 127.0.0.1 only; there is no auth, since it never listens on
anything but loopback — same stance 'sgo ui' already takes.

Reads whatever's on disk rather than regenerating on every request —
run 'sgo generate openapi' first if it doesn't exist yet (same contract
'sgo openapi validate' already has), and again whenever you want the
viewer to pick up route changes; no server restart needed, just a
browser refresh.

Example:
  sgo openapi ui
  sgo openapi ui --port 4749
`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadProjectConfig()
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		if _, err := openapiui.Handler(".", cfg); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("📖 sgo openapi ui running at http://127.0.0.1:%d\n", openapiUIPort)
		if err := openapiui.Serve(".", cfg, openapiUIPort); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	openapiUICmd.Flags().IntVar(&openapiUIPort, "port", 4749, "Port to bind the OpenAPI viewer to")

	openapiCmd.AddCommand(openapiValidateCmd)
	openapiCmd.AddCommand(openapiUICmd)
	rootCmd.AddCommand(openapiCmd)
}
