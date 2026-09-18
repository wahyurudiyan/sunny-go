package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List services in this sgo project",
	Long: `List every service tracked in sgo.yaml, showing whether each stage of
generation has run for it: proto -> contract/gen -> domain entity ->
service implementation.

Example:
  sgo list services`,
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
		fmt.Println("Usage: sgo list services")
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
		fmt.Printf("  - %s\n", name)
		printCheck("proto", filepath.Join("contract", "pb", name+".proto"))
		printCheck("contract/gen", filepath.Join("contract", "gen", name))
		printCheck("domain entity", filepath.Join("internal", "core", "domain", name))
		printCheck("service implementation", filepath.Join("internal", "core", "service", name+"_service.go"))
	}

	return nil
}

func printCheck(label, path string) {
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("    ✅ %s\n", label)
		return
	}
	fmt.Printf("    ❌ %s (missing: %s)\n", label, path)
}

func init() {
	listCmd.AddCommand(listServicesCmd)
	rootCmd.AddCommand(listCmd)
}
