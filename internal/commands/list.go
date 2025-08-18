package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wahyurudiyan/sunny-go/internal/generate"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available services",
	Long: `List all available services in the project based on proto files.
	
This command will show:
- All proto files in api/contract/proto/
- Whether API files have been generated for each service
- Whether service implementation files have been generated

Example:
  sunny list services`,
	Run: func(cmd *cobra.Command, args []string) {
		handleList(cmd, args)
	},
}

var listServicesCmd = &cobra.Command{
	Use:   "services",
	Short: "List all available services",
	Long:  "List all available services based on proto files in the project.",
	Run: func(cmd *cobra.Command, args []string) {
		if err := generate.ListServices(); err != nil {
			fmt.Printf("Error listing services: %v\n", err)
			os.Exit(1)
		}
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate proto files",
	Long: `Validate proto files for syntax and semantic errors.
	
This command uses protoc to validate proto files and report any issues.

Examples:
  sunny validate proto/user.proto
  sunny validate api/contract/proto/user.proto`,
	Run: func(cmd *cobra.Command, args []string) {
		handleValidate(cmd, args)
	},
}

func handleList(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		fmt.Println("Please specify what to list:")
		fmt.Println("  sunny list services")
		return
	}

	switch args[0] {
	case "services":
		if err := generate.ListServices(); err != nil {
			fmt.Printf("Error listing services: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Unknown list type: %s\n", args[0])
		fmt.Println("Available types: services")
	}
}

func handleValidate(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: sunny validate <proto_file>")
		fmt.Println("Example: sunny validate api/contract/proto/user.proto")
		return
	}

	protoPath := args[0]
	if err := generate.ValidateProto(protoPath); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Add list command and its subcommands
	listCmd.AddCommand(listServicesCmd)
	rootCmd.AddCommand(listCmd)

	// Add validate command
	rootCmd.AddCommand(validateCmd)
}
