package generate

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// getModulePath reads the module path from go.mod file
func getModulePath() (string, error) {
	file, err := os.Open("go.mod")
	if err != nil {
		return "", fmt.Errorf("go.mod file not found: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading go.mod: %w", err)
	}

	return "", fmt.Errorf("module declaration not found in go.mod")
}

// HTTPFramework represents the supported HTTP frameworks
type HTTPFramework string

const (
	HTTPFrameworkGin   HTTPFramework = "gin"
	HTTPFrameworkEcho  HTTPFramework = "echo"
	HTTPFrameworkFiber HTTPFramework = "fiber"
)

// GenerateConfig holds configuration for generating files
type GenerateConfig struct {
	ServiceName   string
	HTTPFramework HTTPFramework
	ModulePath    string
}

// GenerateProtoContract generates a proto contract file
func GenerateProtoContract(serviceName string) error {
	if err := validateServiceName(serviceName); err != nil {
		return err
	}

	config := &GenerateConfig{
		ServiceName: serviceName,
	}

	// Create directory structure
	contractDir := filepath.Join("api", "contract", "proto")
	if err := os.MkdirAll(contractDir, 0755); err != nil {
		return fmt.Errorf("failed to create contract directory: %w", err)
	}

	// Generate proto file
	protoPath := filepath.Join(contractDir, serviceName+".proto")
	if err := generateFileFromTemplate(protoPath, ProtoTemplate, config); err != nil {
		return fmt.Errorf("failed to generate proto file: %w", err)
	}

	fmt.Printf("✅ Successfully generated proto contract: %s\n", protoPath)
	fmt.Printf("📝 Next steps:\n")
	fmt.Printf("   sunny generate api %s\n", serviceName)
	fmt.Printf("   sunny generate service %s\n", serviceName)

	return nil
}

// GenerateAPI generates API files from proto and service files if they don't exist
func GenerateAPI(serviceName string) error {
	if err := validateServiceName(serviceName); err != nil {
		return err
	}

	// Check if proto file exists
	protoPath := filepath.Join("api", "contract", "proto", serviceName+".proto")
	if _, err := os.Stat(protoPath); os.IsNotExist(err) {
		return fmt.Errorf("proto file not found: %s. Please run 'sunny generate contract proto %s' first", protoPath, serviceName)
	}

	// Create API directory
	apiDir := filepath.Join("api", serviceName)
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		return fmt.Errorf("failed to create API directory: %w", err)
	}

	// Generate API files using simple templates
	if err := generateAPIFiles(serviceName, protoPath, apiDir); err != nil {
		return fmt.Errorf("failed to generate API files: %w", err)
	}

	fmt.Printf("✅ Successfully generated API files in: %s\n", apiDir)

	// Check if service files exist, if not generate them automatically
	serviceDir := filepath.Join("internal", "service", serviceName)
	serviceFile := filepath.Join(serviceDir, serviceName+".go")

	if _, err := os.Stat(serviceFile); os.IsNotExist(err) {
		fmt.Printf("🔄 Service files don't exist, generating them automatically...\n")

		// Create service directory
		if err := os.MkdirAll(serviceDir, 0755); err != nil {
			return fmt.Errorf("failed to create service directory: %w", err)
		}

		// Get module path from go.mod
		modulePath, err := getModulePath()
		if err != nil {
			fmt.Printf("⚠️ Warning: Could not get module path, service files may need manual adjustment: %v\n", err)
			modulePath = "your-module-path"
		}

		config := &GenerateConfig{
			ServiceName: serviceName,
			ModulePath:  modulePath,
		}

		// Generate service files
		if err := generateServiceFiles(serviceName, serviceDir, config); err != nil {
			fmt.Printf("⚠️ Warning: Failed to generate service files: %v\n", err)
		} else {
			fmt.Printf("✅ Successfully generated service files in: %s\n", serviceDir)
		}
	}

	return nil
}

// GenerateService generates service files from proto
func GenerateService(serviceName string) error {
	if err := validateServiceName(serviceName); err != nil {
		return err
	}

	// Check if proto file exists
	protoPath := filepath.Join("api", "contract", "proto", serviceName+".proto")
	if _, err := os.Stat(protoPath); os.IsNotExist(err) {
		return fmt.Errorf("proto file not found: %s. Please run 'sunny generate contract proto %s' first", protoPath, serviceName)
	}

	// Create service directory
	serviceDir := filepath.Join("internal", "service", serviceName)
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	// Get module path from go.mod
	modulePath, err := getModulePath()
	if err != nil {
		return fmt.Errorf("failed to get module path: %w", err)
	}

	config := &GenerateConfig{
		ServiceName: serviceName,
		ModulePath:  modulePath,
	}

	// Generate service files
	if err := generateServiceFiles(serviceName, serviceDir, config); err != nil {
		return fmt.Errorf("failed to generate service files: %w", err)
	}

	fmt.Printf("✅ Successfully generated service files in: %s\n", serviceDir)
	return nil
}

// ListServices lists all available services based on proto files
func ListServices() error {
	protoDir := filepath.Join("api", "contract", "proto")

	if _, err := os.Stat(protoDir); os.IsNotExist(err) {
		fmt.Println("No proto directory found. Use 'sunny generate contract proto <service_name>' to create services.")
		return nil
	}

	files, err := os.ReadDir(protoDir)
	if err != nil {
		return fmt.Errorf("failed to read proto directory: %w", err)
	}

	var services []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".proto") {
			serviceName := strings.TrimSuffix(file.Name(), ".proto")
			services = append(services, serviceName)
		}
	}

	if len(services) == 0 {
		fmt.Println("No services found. Use 'sunny generate contract proto <service_name>' to create services.")
		return nil
	}

	fmt.Println("Available services:")
	for _, service := range services {
		fmt.Printf("  - %s\n", service)

		// Check if API files exist
		apiDir := filepath.Join("api", service)
		if _, err := os.Stat(apiDir); err == nil {
			fmt.Printf("    ✅ API files generated\n")
		} else {
			fmt.Printf("    ❌ API files not generated (run: sunny generate api %s)\n", service)
		}

		// Check if service files exist
		serviceDir := filepath.Join("internal", "service", service)
		if _, err := os.Stat(serviceDir); err == nil {
			fmt.Printf("    ✅ Service files generated\n")
		} else {
			fmt.Printf("    ❌ Service files not generated (run: sunny generate service %s)\n", service)
		}
	}

	return nil
}

// ValidateProto validates a proto file
func ValidateProto(protoPath string) error {
	if protoPath == "" {
		return fmt.Errorf("proto path cannot be empty")
	}

	// Check if file exists
	if _, err := os.Stat(protoPath); os.IsNotExist(err) {
		return fmt.Errorf("proto file not found: %s", protoPath)
	}

	// Use protoc to validate the proto file
	cmd := exec.Command("protoc", "--proto_path=.", "--descriptor_set_out=/dev/null", protoPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Validation errors:\n%s\n", string(output))
		return fmt.Errorf("proto validation failed: %w", err)
	}

	fmt.Printf("✅ Proto file %s is valid\n", protoPath)
	return nil
}

// validateServiceName validates the service name format
func validateServiceName(name string) error {
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

// generateFileFromTemplate generates a file from a template
func generateFileFromTemplate(filePath, templateContent string, config *GenerateConfig) error {
	// Add template functions
	funcMap := template.FuncMap{
		"title": func(s string) string {
			if len(s) == 0 {
				return s
			}
			return strings.ToUpper(s[:1]) + s[1:]
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
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

	if err := tmpl.Execute(file, config); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// generateServiceFiles generates service implementation files
func generateServiceFiles(serviceName, serviceDir string, config *GenerateConfig) error {
	files := map[string]string{
		serviceName + ".go":      ServiceTemplate,
		serviceName + "_test.go": ServiceTestTemplate,
	}

	for fileName, templateContent := range files {
		filePath := filepath.Join(serviceDir, fileName)
		if err := generateFileFromTemplate(filePath, templateContent, config); err != nil {
			return fmt.Errorf("failed to generate %s: %w", fileName, err)
		}
	}

	return nil
}
