package generate

// Templates for API generation

// ProtoTemplate defines the protocol buffer service template
const ProtoTemplate = `syntax = "proto3";

package {{.ServiceName | lower}};

option go_package = "github.com/yourorg/{{.ServiceName | lower}}/proto";

import "google/api/annotations.proto";
import "google/protobuf/empty.proto";
import "google/protobuf/timestamp.proto";

// {{.ServiceName}} service definition
service {{.ServiceName}}Service {
  // Create a new {{.ServiceName | lower}}
  rpc Create{{.ServiceName}}(Create{{.ServiceName}}Request) returns ({{.ServiceName}}Response) {
    option (google.api.http) = {
      post: "/api/{{.ServiceName | lower}}"
      body: "*"
    };
  }

  // Get {{.ServiceName | lower}} by ID
  rpc Get{{.ServiceName}}(Get{{.ServiceName}}Request) returns ({{.ServiceName}}Response) {
    option (google.api.http) = {
      get: "/api/{{.ServiceName | lower}}/{id}"
    };
  }

  // Update {{.ServiceName | lower}}
  rpc Update{{.ServiceName}}(Update{{.ServiceName}}Request) returns ({{.ServiceName}}Response) {
    option (google.api.http) = {
      put: "/api/{{.ServiceName | lower}}/{id}"
      body: "*"
    };
  }

  // Delete {{.ServiceName | lower}}
  rpc Delete{{.ServiceName}}(Delete{{.ServiceName}}Request) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      delete: "/api/{{.ServiceName | lower}}/{id}"
    };
  }

  // List {{.ServiceName | lower}}s with pagination
  rpc List{{.ServiceName}}s(List{{.ServiceName}}sRequest) returns (List{{.ServiceName}}sResponse) {
    option (google.api.http) = {
      get: "/api/{{.ServiceName | lower}}"
    };
  }
}

// {{.ServiceName}} message
message {{.ServiceName}} {
  string id = 1;
}

// Request messages
message Create{{.ServiceName}}Request {
}

message Get{{.ServiceName}}Request {
  string id = 1;
}

message Update{{.ServiceName}}Request {
  string id = 1;
}

message Delete{{.ServiceName}}Request {
  string id = 1;
}

message List{{.ServiceName}}sRequest {
}

// Response messages
message {{.ServiceName}}Response {
  {{.ServiceName}} data = 1;
}

message List{{.ServiceName}}sResponse {
  repeated {{.ServiceName}} data = 1;
  PaginationMeta pagination = 2;
}

// Pagination metadata
message PaginationMeta {
  int32 page = 1;
  int32 page_size = 2;
  int64 total = 3;
  int32 total_pages = 4;
  bool has_next = 5;
  bool has_prev = 6;
}`

// ServiceTemplate defines the service implementation template
const ServiceTemplate = `package {{.ServiceName}}

import (
	"context"

	"{{.ModulePath}}/api/{{.ServiceName}}"
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
func (s *Service) Create{{.ServiceName | title}}(ctx context.Context, req *{{.ServiceName}}.Create{{.ServiceName | title}}Request) (*{{.ServiceName}}.{{.ServiceName | title}}Response, error) {
	// TODO: Implement create logic
	return nil, status.Error(codes.Unimplemented, "method not implemented")
}

// Get{{.ServiceName | title}} retrieves a {{.ServiceName}} by ID
func (s *Service) Get{{.ServiceName | title}}(ctx context.Context, req *{{.ServiceName}}.Get{{.ServiceName | title}}Request) (*{{.ServiceName}}.{{.ServiceName | title}}Response, error) {
	// TODO: Implement get logic
	return nil, status.Error(codes.Unimplemented, "method not implemented")
}

// List{{.ServiceName | title}}s lists {{.ServiceName}}s
func (s *Service) List{{.ServiceName | title}}s(ctx context.Context, req *{{.ServiceName}}.List{{.ServiceName | title}}sRequest) (*{{.ServiceName}}.List{{.ServiceName | title}}sResponse, error) {
	// TODO: Implement list logic
	return nil, status.Error(codes.Unimplemented, "method not implemented")
}

// Update{{.ServiceName | title}} updates a {{.ServiceName}}
func (s *Service) Update{{.ServiceName | title}}(ctx context.Context, req *{{.ServiceName}}.Update{{.ServiceName | title}}Request) (*{{.ServiceName}}.{{.ServiceName | title}}Response, error) {
	// TODO: Implement update logic
	return nil, status.Error(codes.Unimplemented, "method not implemented")
}

// Delete{{.ServiceName | title}} deletes a {{.ServiceName}}
func (s *Service) Delete{{.ServiceName | title}}(ctx context.Context, req *{{.ServiceName}}.Delete{{.ServiceName | title}}Request) (*{{.ServiceName}}.Empty, error) {
	// TODO: Implement delete logic
	return nil, status.Error(codes.Unimplemented, "method not implemented")
}`

// ServiceTestTemplate defines the service test template
const ServiceTestTemplate = `package {{.ServiceName}}

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
