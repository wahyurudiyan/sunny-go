package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/project"
	"github.com/wahyurudiyan/sunny-go/internal/config"
	"github.com/wahyurudiyan/sunny-go/internal/progress"
	"github.com/wahyurudiyan/sunny-go/internal/wizard"
)

var (
	initModule         string
	initHTTPFramework  string
	initPersistence    string
	initDB             string
	initCache          string
	initSearch         string
	initOpenAPIVersion string
	initOpenAPIFormat  string
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Scaffold a new hexagonal Go service",
	Long: `Scaffold a new project's directory structure and write sgo.yaml.

With no selection flags and a terminal attached, this launches an
interactive wizard for the project name, HTTP framework, persistence
mode, and datastores. Pass any selection flag (or run non-interactively)
to skip the wizard and scaffold directly.

Examples:
  sgo init
  sgo init myservice
  sgo init myservice --http-framework echo --db postgres,mongo --cache redis
`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var name string
		if len(args) == 1 {
			name = args[0]
		}

		opts, err := resolveInitOptions(cmd, name)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		destDir := filepath.Join(".", opts.Name)
		err = progress.Run("Scaffolding "+opts.Name, func() error {
			return project.Scaffold(destDir, *opts)
		})
		if err != nil {
			fmt.Printf("❌ Failed to scaffold project: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Created %s\n\n", destDir)
		fmt.Printf("📁 Project structure:\n")
		fmt.Printf("   %s/\n", opts.Name)
		fmt.Printf("   ├── sgo.yaml                   # project manifest\n")
		fmt.Printf("   ├── contract/pb/                # your .proto files go here\n")
		fmt.Printf("   ├── internal/domain/             # aggregates, value objects, domain events\n")
		fmt.Printf("   ├── internal/application/        # CQRS command/query DTOs + services\n")
		fmt.Printf("   ├── internal/infrastructure/     # HTTP/gRPC transport, persistence, bootstrap\n")
		fmt.Printf("   ├── cmd/%s/\n", opts.Name)
		fmt.Printf("   └── docker/                     # Dockerfile + docker-compose.yml\n")
		fmt.Printf("\n🚀 Next steps:\n")
		fmt.Printf("   cd %s\n", opts.Name)
		fmt.Printf("   sgo generate proto <service>\n")
	},
}

// wizardFlags lists the init flags whose presence means the user wants
// direct, non-interactive control — any one of them skips the wizard.
var wizardFlags = []string{"module", "http-framework", "persistence-mode", "db", "cache", "search", "openapi-version", "openapi-format"}

// resolveInitOptions decides between the interactive wizard and the
// flag-driven path (ARCHITECTURE.md §10): the wizard runs only when no
// selection flag was given and stdin is a terminal.
func resolveInitOptions(cmd *cobra.Command, name string) (*project.Options, error) {
	for _, flag := range wizardFlags {
		if cmd.Flags().Changed(flag) {
			return buildInitOptions(name)
		}
	}

	if !isatty.IsTerminal(os.Stdin.Fd()) {
		return buildInitOptions(name)
	}

	return wizard.Run(name)
}

func buildInitOptions(name string) (*project.Options, error) {
	return project.BuildOptions(
		name,
		initModule,
		config.HTTPFramework(initHTTPFramework),
		config.PersistenceMode(initPersistence),
		splitEngines[config.PersistenceEngine](initDB),
		splitEngines[config.CacheEngine](initCache),
		splitEngines[config.SearchEngine](initSearch),
		config.OpenAPIVersion(initOpenAPIVersion),
		config.OpenAPIFormat(initOpenAPIFormat),
	)
}

// splitEngines parses a comma-separated flag value into a slice of T,
// trimming whitespace and dropping empty entries.
func splitEngines[T ~string](raw string) []T {
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	result := make([]T, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		result = append(result, T(p))
	}

	return result
}

func init() {
	initCmd.Flags().StringVar(&initModule, "module", "", "Go module path (default: project name)")
	initCmd.Flags().StringVar(&initHTTPFramework, "http-framework", "gin", "HTTP framework: gin, echo, chi")
	initCmd.Flags().StringVar(&initPersistence, "persistence-mode", "orm", "Persistence mode: orm, self-managed")
	initCmd.Flags().StringVar(&initDB, "db", "", "Comma-separated datastores: postgres, mysql, mongo")
	initCmd.Flags().StringVar(&initCache, "cache", "", "Comma-separated cache engines: redis")
	initCmd.Flags().StringVar(&initSearch, "search", "", "Comma-separated search engines: elasticsearch")
	initCmd.Flags().StringVar(&initOpenAPIVersion, "openapi-version", string(config.OpenAPIVersion30), "OpenAPI doc version: 3.0, 3.1")
	initCmd.Flags().StringVar(&initOpenAPIFormat, "openapi-format", string(config.OpenAPIFormatYAML), "OpenAPI doc format: yaml, json")

	rootCmd.AddCommand(initCmd)
}
