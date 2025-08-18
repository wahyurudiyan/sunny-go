package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wahyurudiyan/sunny-go/internal/create"
)

var (
	noExample     bool
	httpFramework string
)

var createCmd = &cobra.Command{
	Use:     "create <project_name>",
	Aliases: []string{"new"},
	Short:   "Initialize a new Go web application project",
	Long: `Initialize a new Go web application project with recommended structure and files.

This command creates a complete Go microservice project structure with:
- gRPC service with protobuf definitions
- HTTP server with your chosen framework (fiber, gin, echo)
- Clean architecture layers (models, repository, service, handler)
- Docker setup for local development
- Makefile for common tasks

By default, creates a project with a hello world example API.
Use --no-example to create a basic project layout without example code.

Usage:
  sunny create <project_name> [flags]
  sunny new <project_name> [flags]

Examples:
  sunny create my-api
  sunny create my-api --http-framework gin
  sunny create my-api --no-example
  sunny new auth-service --http-framework echo
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]

		// Validate project name
		if err := create.ValidateProjectName(projectName); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		// Validate HTTP framework
		if err := create.ValidateHTTPFramework(httpFramework); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		config := &create.ProjectConfig{
			ProjectName:   projectName,
			NoExample:     noExample,
			HTTPFramework: httpFramework,
		}

		// Create the project
		if err := create.CreateProject(config); err != nil {
			fmt.Printf("❌ Failed to create project: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	createCmd.Flags().BoolVar(&noExample, "no-example", false, "Create basic project layout without example code")
	createCmd.Flags().StringVar(&httpFramework, "http-framework", "fiber", "HTTP framework to use (fiber, gin, echo)")
	rootCmd.AddCommand(createCmd)
}
