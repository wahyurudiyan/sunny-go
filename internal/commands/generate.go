package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wahyurudiyan/sunny-go/internal/generate"
)

var generateCmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"gen", "g"},
	Short:   "Generate proto contract, api, service, or repository.",
	Long: `Generate various components for your Go application:

- contract proto <service_name>: Generate protobuf contract
- api <service_name>: Generate API files from proto
- service <service_name>: Generate service files from proto

Examples:
  sunny generate contract proto user
  sunny generate api user
  sunny generate service user`,
	Run: func(cmd *cobra.Command, args []string) {
		HandleGenerate(cmd, args)
	},
}

// HandleGenerate handles the generate command and its subcommands
func HandleGenerate(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		fmt.Println("Please specify what to generate: contract, api, or service")
		fmt.Println("\nUsage:")
		fmt.Println("  sunny generate contract proto <service_name>")
		fmt.Println("  sunny generate api <service_name>")
		fmt.Println("  sunny generate service <service_name>")
		return
	}

	switch args[0] {
	case "contract":
		handleContractGenerate(args[1:])
	case "api":
		handleAPIGenerate(args[1:])
	case "service":
		handleServiceGenerate(args[1:])
	default:
		fmt.Printf("Unknown generate type: %s\n", args[0])
		fmt.Println("Available types: contract, api, service")
	}
}

func handleContractGenerate(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: sunny generate contract proto <service_name>")
		return
	}

	if args[0] != "proto" {
		fmt.Printf("Unknown contract type: %s\n", args[0])
		fmt.Println("Available types: proto")
		return
	}

	serviceName := args[1]
	if err := generate.GenerateProtoContract(serviceName); err != nil {
		fmt.Printf("Error generating proto contract: %v\n", err)
		os.Exit(1)
	}
}

func handleAPIGenerate(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: sunny generate api <service_name>")
		return
	}

	serviceName := args[0]
	if err := generate.GenerateAPI(serviceName); err != nil {
		fmt.Printf("Error generating API: %v\n", err)
		os.Exit(1)
	}
}

func handleServiceGenerate(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: sunny generate service <service_name>")
		return
	}

	serviceName := args[0]
	if err := generate.GenerateService(serviceName); err != nil {
		fmt.Printf("Error generating service: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(generateCmd)
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
