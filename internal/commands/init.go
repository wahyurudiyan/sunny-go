package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wahyurudiyan/sunny-go/internal/codegen/project"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

var (
	initModule        string
	initHTTPFramework string
	initPersistence   string
	initDB            string
	initCache         string
	initSearch        string
)

var initCmd = &cobra.Command{
	Use:   "init <project-name>",
	Short: "Scaffold a new hexagonal Go service",
	Long: `Scaffold a new project's directory structure and write sgo.yaml.

This is the non-interactive form (flags only); the interactive selection
wizard is a later phase — see PLAN.md.

Examples:
  sgo init myservice
  sgo init myservice --http-framework echo --db postgres,mongo --cache redis
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		opts, err := buildInitOptions(name)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		destDir := filepath.Join(".", name)
		if err := project.Scaffold(destDir, *opts); err != nil {
			fmt.Printf("❌ Failed to scaffold project: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Created %s\n\n", destDir)
		fmt.Printf("📁 Project structure:\n")
		fmt.Printf("   %s/\n", name)
		fmt.Printf("   ├── sgo.yaml                   # project manifest\n")
		fmt.Printf("   ├── contract/pb/                # your .proto files go here\n")
		fmt.Printf("   ├── internal/core/               # hexagonal core (domain, ports, services)\n")
		fmt.Printf("   ├── internal/adapter/             # HTTP/gRPC + datastore adapters\n")
		fmt.Printf("   ├── cmd/%s/\n", name)
		fmt.Printf("   └── docker/                     # Dockerfile + docker-compose.yml\n")
		fmt.Printf("\n🚀 Next steps:\n")
		fmt.Printf("   cd %s\n", name)
		fmt.Printf("   sgo generate proto <service>\n")
	},
}

func buildInitOptions(name string) (*project.Options, error) {
	if err := project.ValidateName(name); err != nil {
		return nil, err
	}

	module := initModule
	if module == "" {
		module = name
	}

	opts := &project.Options{
		Name:          name,
		Module:        module,
		HTTPFramework: config.HTTPFramework(initHTTPFramework),
		Persistence: config.Persistence{
			Mode:    config.PersistenceMode(initPersistence),
			Engines: splitEngines[config.PersistenceEngine](initDB),
		},
		Cache:  splitEngines[config.CacheEngine](initCache),
		Search: splitEngines[config.SearchEngine](initSearch),
	}

	cfg := config.Config{
		Module:        opts.Module,
		HTTPFramework: opts.HTTPFramework,
		Persistence:   opts.Persistence,
		Cache:         opts.Cache,
		Search:        opts.Search,
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return opts, nil
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

	rootCmd.AddCommand(initCmd)
}
