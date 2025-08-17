package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

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
	if err := generateFileFromTemplate(protoPath, protoTemplate, config); err != nil {
		return fmt.Errorf("failed to generate proto file: %w", err)
	}

	fmt.Printf("✅ Successfully generated proto contract: %s\n", protoPath)
	fmt.Printf("📝 Next steps:\n")
	fmt.Printf("   sunny generate api %s\n", serviceName)
	fmt.Printf("   sunny generate service %s\n", serviceName)

	return nil
}

// GenerateAPI generates API files from proto
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

	// Generate API files using protoc
	if err := generateAPIFiles(serviceName, protoPath, apiDir); err != nil {
		return fmt.Errorf("failed to generate API files: %w", err)
	}

	fmt.Printf("✅ Successfully generated API files in: %s\n", apiDir)
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

	config := &GenerateConfig{
		ServiceName: serviceName,
	}

	// Generate service files
	if err := generateServiceFiles(serviceName, serviceDir, config); err != nil {
		return fmt.Errorf("failed to generate service files: %w", err)
	}

	fmt.Printf("✅ Successfully generated service files in: %s\n", serviceDir)
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

// generateAPIFiles generates API files using protoc
func generateAPIFiles(serviceName, protoPath, apiDir string) error {
	// This will be implemented to call protoc commands
	// For now, create placeholder files

	files := map[string]string{
		serviceName + ".pb.go":      "// Generated protobuf code will be here\npackage " + serviceName + "\n",
		serviceName + "_http.pb.go": "// Generated HTTP gateway code will be here\npackage " + serviceName + "\n",
		serviceName + "_grpc.pb.go": "// Generated gRPC code will be here\npackage " + serviceName + "\n",
	}

	for fileName, content := range files {
		filePath := filepath.Join(apiDir, fileName)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create %s: %w", fileName, err)
		}
	}

	return nil
}

// generateServiceFiles generates service implementation files
func generateServiceFiles(serviceName, serviceDir string, config *GenerateConfig) error {
	files := map[string]string{
		serviceName + ".go":      serviceTemplate,
		serviceName + "_test.go": serviceTestTemplate,
	}

	for fileName, templateContent := range files {
		filePath := filepath.Join(serviceDir, fileName)
		if err := generateFileFromTemplate(filePath, templateContent, config); err != nil {
			return fmt.Errorf("failed to generate %s: %w", fileName, err)
		}
	}

	return nil
}

// Template definitions
const protoTemplate = `syntax = "proto3";

package {{.ServiceName}};

option go_package = "github.com/yourorg/{{.ServiceName}}/api/{{.ServiceName}}";

import "google/api/annotations.proto";
import "google/protobuf/empty.proto";
import "google/protobuf/timestamp.proto";

// {{.ServiceName | title}} service definition
service {{.ServiceName | title}}Service {
  // Create a new {{.ServiceName}}
  rpc Create{{.ServiceName | title}}(Create{{.ServiceName | title}}Request) returns ({{.ServiceName | title}}Response) {
    option (google.api.http) = {
      post: "/api/v1/{{.ServiceName}}s"
      body: "*"
    };
  }

  // Get a {{.ServiceName}} by ID
  rpc Get{{.ServiceName | title}}(Get{{.ServiceName | title}}Request) returns ({{.ServiceName | title}}Response) {
    option (google.api.http) = {
      get: "/api/v1/{{.ServiceName}}s/{id}"
    };
  }

  // List {{.ServiceName}}s
  rpc List{{.ServiceName | title}}s(List{{.ServiceName | title}}sRequest) returns (List{{.ServiceName | title}}sResponse) {
    option (google.api.http) = {
      get: "/api/v1/{{.ServiceName}}s"
    };
  }

  // Update a {{.ServiceName}}
  rpc Update{{.ServiceName | title}}(Update{{.ServiceName | title}}Request) returns ({{.ServiceName | title}}Response) {
    option (google.api.http) = {
      put: "/api/v1/{{.ServiceName}}s/{id}"
      body: "*"
    };
  }

  // Delete a {{.ServiceName}}
  rpc Delete{{.ServiceName | title}}(Delete{{.ServiceName | title}}Request) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      delete: "/api/v1/{{.ServiceName}}s/{id}"
    };
  }
}

// {{.ServiceName | title}} message
message {{.ServiceName | title}} {
  string id = 1;
  string name = 2;
  string description = 3;
  google.protobuf.Timestamp created_at = 4;
  google.protobuf.Timestamp updated_at = 5;
}

// Request messages
message Create{{.ServiceName | title}}Request {
  string name = 1;
  string description = 2;
}

message Get{{.ServiceName | title}}Request {
  string id = 1;
}

message List{{.ServiceName | title}}sRequest {
  int32 page = 1;
  int32 page_size = 2;
}

message Update{{.ServiceName | title}}Request {
  string id = 1;
  string name = 2;
  string description = 3;
}

message Delete{{.ServiceName | title}}Request {
  string id = 1;
}

// Response messages
message {{.ServiceName | title}}Response {
  {{.ServiceName | title}} {{.ServiceName}} = 1;
}

message List{{.ServiceName | title}}sResponse {
  repeated {{.ServiceName | title}} {{.ServiceName}}s = 1;
  int32 total = 2;
  int32 page = 3;
  int32 page_size = 4;
}`

const serviceTemplate = `package {{.ServiceName}}

import (
	"context"
	"fmt"

	"go.uber.org/fx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service represents the {{.ServiceName}} service
type Service struct {
	// Add your dependencies here
}

// ServiceParams defines the dependencies for the service
type ServiceParams struct {
	fx.In
	// Add your dependencies here
}

// New creates a new {{.ServiceName}} service
func New(params ServiceParams) *Service {
	return &Service{
		// Initialize dependencies
	}
}

// Create{{.ServiceName | title}} creates a new {{.ServiceName}}
func (s *Service) Create{{.ServiceName | title}}(ctx context.Context, req *Create{{.ServiceName | title}}Request) (*{{.ServiceName | title}}Response, error) {
	// TODO: Implement create logic
	return nil, status.Error(codes.Unimplemented, "method not implemented")
}

// Get{{.ServiceName | title}} retrieves a {{.ServiceName}} by ID
func (s *Service) Get{{.ServiceName | title}}(ctx context.Context, req *Get{{.ServiceName | title}}Request) (*{{.ServiceName | title}}Response, error) {
	// TODO: Implement get logic
	return nil, status.Error(codes.Unimplemented, "method not implemented")
}

// List{{.ServiceName | title}}s lists {{.ServiceName}}s
func (s *Service) List{{.ServiceName | title}}s(ctx context.Context, req *List{{.ServiceName | title}}sRequest) (*List{{.ServiceName | title}}sResponse, error) {
	// TODO: Implement list logic
	return nil, status.Error(codes.Unimplemented, "method not implemented")
}

// Update{{.ServiceName | title}} updates a {{.ServiceName}}
func (s *Service) Update{{.ServiceName | title}}(ctx context.Context, req *Update{{.ServiceName | title}}Request) (*{{.ServiceName | title}}Response, error) {
	// TODO: Implement update logic
	return nil, status.Error(codes.Unimplemented, "method not implemented")
}

// Delete{{.ServiceName | title}} deletes a {{.ServiceName}}
func (s *Service) Delete{{.ServiceName | title}}(ctx context.Context, req *Delete{{.ServiceName | title}}Request) (*emptypb.Empty, error) {
	// TODO: Implement delete logic
	return nil, status.Error(codes.Unimplemented, "method not implemented")
}`

const serviceTestTemplate = `package {{.ServiceName}}

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestService_Create{{.ServiceName | title}}(t *testing.T) {
	service := &Service{}
	
	req := &Create{{.ServiceName | title}}Request{
		Name:        "Test {{.ServiceName | title}}",
		Description: "Test Description",
	}

	_, err := service.Create{{.ServiceName | title}}(context.Background(), req)
	
	// For now, expect unimplemented error
	assert.Error(t, err)
}

func TestService_Get{{.ServiceName | title}}(t *testing.T) {
	service := &Service{}
	
	req := &Get{{.ServiceName | title}}Request{
		Id: "test-id",
	}

	_, err := service.Get{{.ServiceName | title}}(context.Background(), req)
	
	// For now, expect unimplemented error
	assert.Error(t, err)
}

func TestService_List{{.ServiceName | title}}s(t *testing.T) {
	service := &Service{}
	
	req := &List{{.ServiceName | title}}sRequest{
		Page:     1,
		PageSize: 10,
	}

	_, err := service.List{{.ServiceName | title}}s(context.Background(), req)
	
	// For now, expect unimplemented error
	assert.Error(t, err)
}

func TestService_Update{{.ServiceName | title}}(t *testing.T) {
	service := &Service{}
	
	req := &Update{{.ServiceName | title}}Request{
		Id:          "test-id",
		Name:        "Updated {{.ServiceName | title}}",
		Description: "Updated Description",
	}

	_, err := service.Update{{.ServiceName | title}}(context.Background(), req)
	
	// For now, expect unimplemented error
	assert.Error(t, err)
}

func TestService_Delete{{.ServiceName | title}}(t *testing.T) {
	service := &Service{}
	
	req := &Delete{{.ServiceName | title}}Request{
		Id: "test-id",
	}

	_, err := service.Delete{{.ServiceName | title}}(context.Background(), req)
	
	// For now, expect unimplemented error
	assert.Error(t, err)
}`
