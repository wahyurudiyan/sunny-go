package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/wahyurudiyan/sunny-go/internal/codegen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/openapigen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
	"github.com/wahyurudiyan/sunny-go/internal/config"
)

var generateCmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"gen", "g"},
	Short:   "Generate proto contracts and code for a service",
}

var generateProtoCmd = &cobra.Command{
	Use:   "proto <name>",
	Short: "Scaffold contract/pb/<name>.proto",
	Long: `Scaffold a starter proto file (a CRUD-shaped service plus its
messages) at contract/pb/<name>.proto. Edit it by hand, then run
'sgo generate code <name>' to generate everything that derives from it.

Refuses to overwrite an existing proto file.

Example:
  sgo generate proto user
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		cfg, err := loadProjectConfig()
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		if err := proto.GenerateStub("contract/pb", name, cfg.Module); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Created contract/pb/%s.proto\n\n", name)
		fmt.Printf("📝 Next steps:\n")
		fmt.Printf("   edit contract/pb/%s.proto\n", name)
		fmt.Printf("   sgo generate code %s\n", name)
	},
}

var generateCodeCmd = &cobra.Command{
	Use:   "code <name>",
	Short: "Generate the DDD domain/application/infrastructure layers and wire types for <name>",
	Long: `Compile contract/pb/<name>.proto and (re)generate everything that
derives from it: contract/gen (protoc-gen-go/protoc-gen-go-grpc output),
the domain aggregate and its repository port, the CQRS command/query
DTOs and application service, the wire<->domain and wire<->application
mappers, HTTP routes for the project's chosen framework, and the gRPC
server adapter.

Safe to run repeatedly: hand-written business logic in the owned files
(internal/domain/<name>/aggregate.go and
internal/application/<name>/service.go) is created once and never
overwritten — see ARCHITECTURE.md §6/§17.

Example:
  sgo generate code user
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		cfg, err := loadProjectConfig()
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		if err := codegen.GenerateCode(".", name, cfg); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Generated code for %s\n\n", name)
		fmt.Printf("   contract/gen/%s/               # wire types (generated, do not edit)\n", name)
		fmt.Printf("   internal/domain/%s/            # aggregate, value objects, domain events\n", name)
		fmt.Printf("   internal/application/%s/       # CQRS command/query DTOs + service — implement it here\n", name)
		fmt.Printf("   internal/infrastructure/       # transport (HTTP/gRPC), persistence, bootstrap\n")
		fmt.Printf("\n📝 Next steps:\n")
		fmt.Printf("   implement internal/application/%s/service.go\n", name)
		fmt.Printf("   sgo list endpoints %s          # see the HTTP routes this service now serves\n", name)
		fmt.Printf("   sgo generate openapi           # generate docs/openapi.yaml for every registered service\n")
	},
}

var (
	generateOpenAPIVersion string
	generateOpenAPIFormat  string
)

var generateOpenAPICmd = &cobra.Command{
	Use:   "openapi",
	Short: "Generate docs/openapi.<ext> from every registered service",
	Long: `Recompile every service in sgo.yaml's services list, derive their HTTP
routes the same way 'sgo generate code' does (ARCHITECTURE.md §13), and
write a fresh docs/openapi.yaml or docs/openapi.json describing the
whole project's REST API. Always fully overwritten — never hand-edited,
same contract as contract/gen.

Validates the document it just wrote against the real OpenAPI meta-schema
before reporting success (the same check 'sgo openapi validate' runs on
demand) — if this fails, it's a bug in sgo itself.

--version/--format override sgo.yaml's openapi.version/openapi.format
for this run only, without changing the persisted selection.

Example:
  sgo generate openapi
  sgo generate openapi --version 3.1 --format json
`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadProjectConfig()
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		if cmd.Flags().Changed("version") {
			cfg.OpenAPI.Version = config.OpenAPIVersion(generateOpenAPIVersion)
		}
		if cmd.Flags().Changed("format") {
			cfg.OpenAPI.Format = config.OpenAPIFormat(generateOpenAPIFormat)
		}

		path, err := openapigen.Generate(cfg, ".")
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Generated %s\n", path)
	},
}

// loadProjectConfig loads sgo.yaml from the current directory, with an
// error message pointing at `sgo init` if this isn't an sgo project.
func loadProjectConfig() (*config.Config, error) {
	cfg, err := config.Load(".")
	if err != nil {
		return nil, fmt.Errorf("not an sgo project (no sgo.yaml found in the current directory): %w", err)
	}
	return cfg, nil
}

func init() {
	generateOpenAPICmd.Flags().StringVar(&generateOpenAPIVersion, "version", "", "Override sgo.yaml's openapi.version for this run: 3.0, 3.1")
	generateOpenAPICmd.Flags().StringVar(&generateOpenAPIFormat, "format", "", "Override sgo.yaml's openapi.format for this run: yaml, json")

	generateCmd.AddCommand(generateProtoCmd)
	generateCmd.AddCommand(generateCodeCmd)
	generateCmd.AddCommand(generateOpenAPICmd)
	rootCmd.AddCommand(generateCmd)
}
