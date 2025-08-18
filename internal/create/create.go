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
	ProjectName   string
	ModulePath    string
	HTTPFramework string
	NoExample     bool
}

// ValidateProjectName validates the project name format
func ValidateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	if strings.Contains(name, " ") {
		return fmt.Errorf("project name cannot contain spaces")
	}

	if strings.ContainsAny(name, "/\\:*?\"<>|") {
		return fmt.Errorf("project name contains invalid characters")
	}

	return nil
}

// ValidateHTTPFramework validates the HTTP framework choice
func ValidateHTTPFramework(framework string) error {
	validFrameworks := []string{"fiber", "gin", "echo"}

	for _, valid := range validFrameworks {
		if framework == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid HTTP framework '%s'. Valid options: %s", framework, strings.Join(validFrameworks, ", "))
}

// CreateProject creates a new Go API project with the specified configuration
func CreateProject(config *ProjectConfig) error {
	if config.ProjectName == "" {
		return fmt.Errorf("project name is required")
	}

	// Set module path to just the project name
	if config.ModulePath == "" {
		config.ModulePath = config.ProjectName
	}

	// Create project directory
	projectPath := filepath.Join(".", config.ProjectName)
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

	fmt.Printf("✅ Successfully created project '%s' at %s\n", config.ProjectName, projectPath)
	fmt.Printf("\n📁 Project structure:\n")
	fmt.Printf("   %s/\n", config.ProjectName)
	fmt.Printf("   ├── api/contract/proto/        # Protocol buffer definitions\n")
	fmt.Printf("   ├── api/                       # Generated API code\n")
	fmt.Printf("   ├── server/                    # gRPC and HTTP servers\n")
	fmt.Printf("   ├── internal/service/          # Business logic\n")
	fmt.Printf("   ├── cmd/                       # Application entrypoints\n")
	fmt.Printf("   ├── docker/                    # Docker configurations\n")
	fmt.Printf("   ├── go.mod                     # Go module file\n")
	fmt.Printf("   ├── Makefile                   # Build automation\n")
	fmt.Printf("   └── README.md                  # Project documentation\n")

	fmt.Printf("\n🚀 Next steps:\n")
	fmt.Printf("   cd %s\n", config.ProjectName)

	if !config.NoExample {
		fmt.Printf("   make build\n")
		fmt.Printf("   make run\n")
	} else {
		fmt.Printf("   sunny generate contract proto user\n")
		fmt.Printf("   sunny generate api user\n")
		fmt.Printf("   make build\n")
	}

	return nil
}

// createDirectoryStructure creates the basic directory structure for the project
func createDirectoryStructure(projectPath string) error {
	dirs := []string{
		"cmd",
		"internal/service",
		"internal/handler",
		"internal/middleware",
		"internal/config",
		"internal/repository",
		"internal/models",
		"api/contract/proto",
		"server",
		"docker",
		"scripts",
		"bin",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(projectPath, dir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// generateProjectFiles generates project files from templates
func generateProjectFiles(projectPath string, config *ProjectConfig) error {
	// Define file templates based on whether example is requested
	var files map[string]string

	if config.NoExample {
		// Basic project files without example
		files = map[string]string{
			"go.mod":                                 templates.GoModTemplate,
			"Makefile":                               templates.MakefileTemplate,
			"docker/Dockerfile":                      templates.DockerfileTemplate,
			"docker/docker-compose.yml":              templates.DockerComposeTemplate,
			"cmd/" + config.ProjectName + "/main.go": templates.MainTemplate,
		}
	} else {
		// Project with example user service
		files = map[string]string{
			"go.mod":                                 templates.GoModTemplate,
			"Makefile":                               templates.MakefileTemplate,
			"docker/Dockerfile":                      templates.DockerfileTemplate,
			"docker/docker-compose.yml":              templates.DockerComposeTemplate,
			"cmd/" + config.ProjectName + "/main.go": templates.MainTemplate,
			"internal/config/config.go":              templates.ConfigTemplate,
			"internal/middleware/logging.go":         templates.LoggingMiddlewareTemplate,
		}
	}

	// Generate files from templates
	for fileName, templateContent := range files {
		filePath := filepath.Join(projectPath, fileName)

		// Create directory for the file if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", fileName, err)
		}

		if err := generateFileFromTemplate(filePath, templateContent, config); err != nil {
			return fmt.Errorf("failed to generate %s: %w", fileName, err)
		}
	}

	return nil
}

// generateFileFromTemplate generates a file from a template
func generateFileFromTemplate(filePath, templateContent string, config *ProjectConfig) error {
	// Add template functions
	funcMap := template.FuncMap{
		"title": func(s string) string {
			if len(s) == 0 {
				return s
			}
			return strings.ToUpper(s[:1]) + s[1:]
		},
	}

	tmpl, err := template.New(filepath.Base(filePath)).Funcs(funcMap).Parse(templateContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Create a template data struct that includes both ProjectName and ServiceName for compatibility
	templateData := struct {
		ProjectName   string
		ServiceName   string // alias for ProjectName for template compatibility
		ModulePath    string
		HTTPFramework string
		NoExample     bool
	}{
		ProjectName:   config.ProjectName,
		ServiceName:   config.ProjectName, // Use ProjectName as ServiceName
		ModulePath:    config.ModulePath,
		HTTPFramework: config.HTTPFramework,
		NoExample:     config.NoExample,
	}

	if err := tmpl.Execute(file, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
} // ValidateServiceName validates the service name format (kept for backward compatibility)
func ValidateServiceName(name string) error {
	return ValidateProjectName(name)
}
