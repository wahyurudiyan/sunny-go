# test-setup

A gRPC and HTTP API service generated with sunny-go.

## Project Structure

```
test-setup/
├── cmd/test-setup/           # Application entry point
├── internal/
│   ├── generator/    # Code generation logic
│   ├── plugin/       # Custom protoc plugins
│   └── service/      # Service implementations
├── proto/            # Protocol buffer definitions
├── api/              # Generated service APIs
├── generated/        # Generated protobuf code
├── bin/              # Compiled binaries
└── Makefile          # Build automation
```

## Quick Start

1. **Generate a service:**
   ```bash
   sunny generate contract proto user
   sunny generate api user
   sunny generate service user
   ```

2. **Build and run:**
   ```bash
   make build
   make run
   ```

## Available Commands

- `sunny generate contract proto <service>` - Generate proto contract
- `sunny generate api <service>` - Generate API files (.pb.go, _grpc.pb.go, _http.pb.go)
- `sunny generate service <service>` - Generate service implementation
- `sunny list services` - List all available services
- `sunny validate <proto_file>` - Validate proto file

## Development

- **Build:** `make build`
- **Test:** `make test`
- **Generate proto:** `make proto`
- **Clean:** `make clean`
- **Dev server:** `make dev`

## API Endpoints

The service provides both gRPC and HTTP endpoints:

- **gRPC:** localhost:9090
- **HTTP:** localhost:8080

### Example API Routes

For a service named "user":
- `POST /api/v1/users` - Create user
- `GET /api/v1/users/{id}` - Get user
- `GET /api/v1/users` - List users
- `PUT /api/v1/users/{id}` - Update user  
- `DELETE /api/v1/users/{id}` - Delete user

## Protobuf Style Guide

Follow these conventions for your .proto files:

```protobuf
syntax = "proto3";

package myservice;

option go_package = "github.com/yourorg/myservice/api/myservice";

import "google/api/annotations.proto";

service MyService {
  rpc CreateItem(CreateItemRequest) returns (ItemResponse) {
    option (google.api.http) = {
      post: "/api/v1/items"
      body: "*"
    };
  }
}
```
