package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/wahyurudiyan/sunny-go/internal/codegen"
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
	Short: "Generate the hexagonal core and wire types for <name>",
	Long: `Compile contract/pb/<name>.proto and (re)generate everything that
derives from it: contract/gen (protoc-gen-go/protoc-gen-go-grpc output),
the domain entity, the usecase/repository ports, the service skeleton,
and the wire<->domain mapper.

Safe to run repeatedly: hand-written business logic in the owned files
(internal/core/domain/<name>/<name>.go and
internal/core/service/<name>_service.go) is created once and never
overwritten — see ARCHITECTURE.md §6.

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
		fmt.Printf("   contract/gen/%s/             # wire types (generated, do not edit)\n", name)
		fmt.Printf("   internal/core/domain/%s/     # domain entity\n", name)
		fmt.Printf("   internal/core/port/          # usecase + repository interfaces\n")
		fmt.Printf("   internal/core/service/       # service skeleton — implement it here\n")
		fmt.Printf("   internal/adapter/mapper/     # wire<->domain mapper (generated)\n")
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
	generateCmd.AddCommand(generateProtoCmd)
	generateCmd.AddCommand(generateCodeCmd)
	rootCmd.AddCommand(generateCmd)
}
