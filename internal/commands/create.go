package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wahyurudiyan/sunny-go/internal/create"
)

var createCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"new"},
	Short:   "Initialize a new Go web application project.",
	Long: `Initialize a new Go web application project with recommended structure and files.

This command creates a complete Go microservice project structure with:
- gRPC service with protobuf definitions
- Clean architecture layers (models, repository, service, handler)
- Database integration with PostgreSQL
- Docker setup for local development
- Makefile for common tasks
- Database migrations

Usage:
  sunny create <service_name>
  sunny new <service_name>

Examples:
  sunny create user-service
  sunny new auth-service
  sunny create payment-api
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := args[0]

		// Validate service name
		if err := create.ValidateServiceName(serviceName); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		// Create the project
		if err := create.CreateProject(serviceName); err != nil {
			fmt.Printf("❌ Failed to create project: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}
