package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/wahyurudiyan/sunny-go/internal/codegen"
	"github.com/wahyurudiyan/sunny-go/internal/codegen/httpgen"
	sgoproto "github.com/wahyurudiyan/sunny-go/internal/codegen/proto"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List services or endpoints in this sgo project",
	Long: `List every service tracked in sgo.yaml, showing whether each stage of
generation has run for it (proto -> contract/gen -> domain entity ->
service implementation), or every HTTP endpoint they derive.

Example:
  sgo list services
  sgo list endpoints`,
	Run: func(cmd *cobra.Command, args []string) {
		handleList(cmd, args)
	},
}

var listServicesCmd = &cobra.Command{
	Use:   "services",
	Short: "List services tracked in sgo.yaml",
	Run: func(cmd *cobra.Command, args []string) {
		if err := listServices(); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func handleList(cmd *cobra.Command, args []string) {
	if len(args) == 0 || args[0] != "services" {
		fmt.Println("Usage: sgo list services | sgo list endpoints [service]")
		return
	}

	if err := listServices(); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}
}

func listServices() error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	if len(cfg.Services) == 0 {
		fmt.Println("No services yet. Run `sgo generate proto <name>` to create one.")
		return nil
	}

	fmt.Println("Services:")
	for _, name := range cfg.Services {
		s := codegen.Status(".", name)
		fmt.Printf("  - %s\n", name)
		printCheck("proto", s.Proto)
		printCheck("contract/gen", s.ContractGen)
		printCheck("domain entity", s.DomainEntity)
		printCheck("service implementation", s.ServiceImpl)
	}

	return nil
}

func printCheck(label string, ok bool) {
	if ok {
		fmt.Printf("    ✅ %s\n", label)
		return
	}
	fmt.Printf("    ❌ %s\n", label)
}

var listEndpointsCmd = &cobra.Command{
	Use:   "endpoints [service]",
	Short: "List HTTP endpoints derived from registered services",
	Long: `List every HTTP route currently derived for this project's registered
services (all of them with no argument, one with it): method, path, and
the RPC it comes from.

Reuses the exact same route derivation (internal/codegen/httpgen.BuildRoutes)
the generated HTTP adapter's own routes file and "sgo generate openapi"
already use — this prints what the generated adapter actually serves,
not a second guess at it.

Example:
  sgo list endpoints
  sgo list endpoints user`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var name string
		if len(args) == 1 {
			name = args[0]
		}
		if err := listEndpoints(name); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func listEndpoints(name string) error {
	cfg, err := loadProjectConfig()
	if err != nil {
		return err
	}

	services := cfg.Services
	if name != "" {
		found := false
		for _, s := range services {
			if s == name {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("service %q not found in sgo.yaml (run `sgo generate code %s` first)", name, name)
		}
		services = []string{name}
	}

	if len(services) == 0 {
		fmt.Println("No services yet. Run `sgo generate proto <name>` then `sgo generate code <name>`.")
		return nil
	}

	colonStyle, err := httpgen.ColonStyle(cfg.HTTPFramework)
	if err != nil {
		return err
	}

	protoDir := filepath.Join(".", "contract", "pb")
	printed := false

	for _, svc := range services {
		fd, err := sgoproto.Compile(protoDir, svc+".proto")
		if err != nil {
			return fmt.Errorf("compiling %s.proto: %w", svc, err)
		}

		file, err := sgoproto.Build(fd)
		if err != nil {
			return fmt.Errorf("building IR for %s: %w", svc, err)
		}

		routes, err := httpgen.BuildRoutes(fd, file, svc)
		if err != nil {
			return fmt.Errorf("deriving routes for %s: %w", svc, err)
		}
		if len(routes) == 0 {
			continue
		}

		printed = true
		fmt.Printf("%s:\n", svc)
		for _, r := range routes {
			fmt.Printf("  %-6s %-30s %s\n", r.Verb, r.FullPath(colonStyle), r.Method.Name)
		}
	}

	if !printed {
		fmt.Println("No endpoints yet. Run `sgo generate code <name>` for a service whose proto declares a service block.")
	}

	return nil
}

func init() {
	listCmd.AddCommand(listServicesCmd)
	listCmd.AddCommand(listEndpointsCmd)
	rootCmd.AddCommand(listCmd)
}
