package create

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/wahyurudiyan/sunny-go/internal/templates"
)

// ProjectConfig holds configuration for creating a new project
type ProjectConfig struct {
	ServiceName   string
	ModulePath    string
	HTTPFramework string
}

// CreateProject creates a new Go API project with the specified structure
func CreateProject(serviceName string) error {
	if serviceName == "" {
		return fmt.Errorf("service name is required")
	}

	// Ask user to select HTTP framework
	httpFramework, err := selectHTTPFramework()
	if err != nil {
		return fmt.Errorf("failed to select HTTP framework: %w", err)
	}

	config := &ProjectConfig{
		ServiceName:   serviceName,
		ModulePath:    fmt.Sprintf("github.com/yourorg/%s", serviceName),
		HTTPFramework: httpFramework,
	}

	// Create project directory
	projectPath := filepath.Join(".", serviceName)
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Create directory structure
	if err := createDirectoryStructure(projectPath); err != nil {
		return fmt.Errorf("failed to create directory structure: %w", err)
	}

	// Generate files from templates
	if err := generateProjectFiles(projectPath, config); err != nil {
		return fmt.Errorf("failed to generate project files: %w", err)
	}

	fmt.Printf("✅ Successfully created project '%s' at %s\n", serviceName, projectPath)
	fmt.Printf("📁 Project structure:\n")
	fmt.Printf("   %s/\n", serviceName)
	fmt.Printf("   ├── api/contract/proto/        # Protocol buffer definitions\n")
	fmt.Printf("   ├── api/                       # Generated API code\n")
	fmt.Printf("   ├── server/                    # gRPC and HTTP servers\n")
	fmt.Printf("   ├── internal/                  # Private application code\n")
	fmt.Printf("   ├── pkg/                       # Public packages\n")
	fmt.Printf("   ├── cmd/                       # Application entry points\n")
	fmt.Printf("   ├── migrations/                # Database migrations\n")
	fmt.Printf("   ├── docker-compose.yml         # Local development setup\n")
	fmt.Printf("   ├── Dockerfile\n")
	fmt.Printf("   ├── Makefile\n")
	fmt.Printf("   └── go.mod\n")
	fmt.Printf("\n🚀 Next steps:\n")
	fmt.Printf("   cd %s\n", serviceName)
	fmt.Printf("   make build\n")
	fmt.Printf("   make run\n")
	fmt.Printf("\n🔧 Generate components:\n")
	fmt.Printf("   sunny generate contract proto <service_name>\n")
	fmt.Printf("   sunny generate api <service_name>\n")
	fmt.Printf("   sunny generate service <service_name>\n")

	return nil
}

// createDirectoryStructure creates all necessary directories
func createDirectoryStructure(projectPath string) error {
	dirs := []string{
		"api/contract/proto",
		"api",
		"internal/models",
		"internal/repository",
		"internal/service",
		"internal/handler",
		"internal/middleware",
		"internal/config",
		"pkg/database",
		"pkg/logger",
		"cmd",
		"server",
		"migrations",
		"test",
		"test/mocks",
		"test/fixtures",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(projectPath, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// generateProjectFiles creates all template files
func generateProjectFiles(projectPath string, config *ProjectConfig) error {
	files := map[string]string{
		"go.mod": templates.GoModTemplate,
		// "README.md":                            templates.ReadmeTemplate,
		"Makefile":                             templates.MakefileTemplate,
		"Dockerfile":                           templates.DockerfileTemplate,
		"docker-compose.yml":                   templates.DockerComposeTemplate,
		"cmd/main.go":                          templates.MainTemplate,
		"server/grpc.go":                       templates.GRPCServerTemplate,
		"server/http.go":                       templates.HTTPServerTemplate,
		"api/user.go":                          templates.APIUserTemplate,
		"api/contract/proto/user.proto":        templates.UserProtoTemplate,
		"internal/config/config.go":            templates.ConfigTemplate,
		"internal/models/user.go":              templates.UserModelTemplate,
		"internal/repository/user.go":          templates.UserRepositoryTemplate,
		"internal/service/user.go":             templates.UserServiceTemplate,
		"internal/handler/user.go":             templates.UserHandlerTemplate,
		"internal/middleware/auth.go":          templates.AuthMiddlewareTemplate,
		"internal/middleware/logging.go":       templates.LoggingMiddlewareTemplate,
		"pkg/database/postgres.go":             templates.DatabaseTemplate,
		"pkg/logger/logger.go":                 templates.LoggerTemplate,
		"migrations/001_create_users.up.sql":   templates.MigrationUpTemplate,
		"migrations/001_create_users.down.sql": templates.MigrationDownTemplate,
		// Unit test files
		"internal/models/user_test.go":     templates.UserModelTestTemplate,
		"internal/repository/user_test.go": templates.UserRepositoryTestTemplate,
		"internal/service/user_test.go":    templates.UserServiceTestTemplate,
		"internal/handler/user_test.go":    templates.UserHandlerTestTemplate,
		"api/user_test.go":                 templates.APIUserHandlerTestTemplate,
		"pkg/database/postgres_test.go":    templates.DatabaseTestTemplate,
		"pkg/logger/logger_test.go":        templates.LoggerTestTemplate,
		"test/mocks/repository.go":         templates.MockUserRepositoryTemplate,
		"test/mocks/service.go":            templates.MockUserServiceTemplate,
		"test/mocks/logger.go":             templates.MockLoggerTemplate,
		"test/fixtures/user.go":            templates.UserFixtureTemplate,
		"test/fixtures/helpers.go":         templates.TestHelpersTemplate,
	}

	for filePath, templateContent := range files {
		fullPath := filepath.Join(projectPath, filePath)

		// Ensure directory exists
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", filePath, err)
		}

		// Parse and execute template
		tmpl, err := template.New(filePath).Parse(templateContent)
		if err != nil {
			return fmt.Errorf("failed to parse template for %s: %w", filePath, err)
		}

		file, err := os.Create(fullPath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", filePath, err)
		}
		defer file.Close()

		if err := tmpl.Execute(file, config); err != nil {
			return fmt.Errorf("failed to execute template for %s: %w", filePath, err)
		}
	}

	return nil
}

// ValidateServiceName validates the service name format
func ValidateServiceName(name string) error {
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	if strings.Contains(name, " ") {
		return fmt.Errorf("service name cannot contain spaces")
	}

	if strings.ContainsAny(name, "/\\:*?\"<>|") {
		return fmt.Errorf("service name contains invalid characters")
	}

	return nil
}

// selectHTTPFramework prompts user to select an HTTP framework
func selectHTTPFramework() (string, error) {
	fmt.Println("Select HTTP framework:")
	fmt.Println("1. Gin")
	fmt.Println("2. Echo")
	fmt.Println("3. Fiber")
	fmt.Print("Enter your choice (1-3, default: 1): ")

	var choice string
	fmt.Scanln(&choice)

	switch choice {
	case "1", "":
		return "gin", nil
	case "2":
		return "echo", nil
	case "3":
		return "fiber", nil
	default:
		return "gin", nil // Default to gin
	}
}
