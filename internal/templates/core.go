package templates

// Core application templates

// Main application template
const MainTemplate = `package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"{{.ModulePath}}/internal/config"
	"{{.ModulePath}}/internal/handler"
	"{{.ModulePath}}/internal/middleware"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/service"
	"{{.ModulePath}}/pkg/database"
	"{{.ModulePath}}/pkg/logger"
	"{{.ModulePath}}/server"
)

func main() {
	app := fx.New(
		// Configuration
		fx.Provide(config.Load),
		
		// Infrastructure
		fx.Provide(logger.New),
		fx.Provide(database.Connect),
		
		// Repositories
		fx.Provide(repository.NewUserRepository),
		
		// Services
		fx.Provide(service.NewUserService),
		
		// Handlers
		fx.Provide(handler.NewUserHandler),
		
		// Middleware
		fx.Provide(middleware.NewAuthMiddleware),
		fx.Provide(middleware.NewLoggingMiddleware),
		
		// Servers
		fx.Provide(server.NewGRPCServer),
		fx.Provide(server.NewHTTPServer),
		
		// Lifecycle hooks
		fx.Invoke(registerHooks),
	)

	// Handle shutdown gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	if err := app.Start(ctx); err != nil {
		os.Exit(1)
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	if err := app.Stop(ctx); err != nil {
		os.Exit(1)
	}
}

// registerHooks registers lifecycle hooks for the servers
func registerHooks(
	lifecycle fx.Lifecycle,
	grpcServer *server.GRPCServer,
	httpServer *server.HTTPServer,
	logger *zap.Logger,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Start GRPC server
			go func() {
				if err := grpcServer.Start(); err != nil {
					logger.Error("GRPC server failed", zap.Error(err))
				}
			}()
			
			// Start HTTP server
			go func() {
				if err := httpServer.Start(); err != nil {
					logger.Error("HTTP server failed", zap.Error(err))
				}
			}()
			
			logger.Info("Servers started successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down servers...")
			
			// Stop servers gracefully
			grpcServer.Stop()
			httpServer.Stop(ctx)
			
			logger.Info("Servers stopped successfully")
			return nil
		},
	})
}`

// Go module template
const GoModTemplate = `// Go module template
const GoModTemplate = ` + "`" + `module {{.ModulePath}}

go 1.21

require (
	github.com/lib/pq v1.10.9
	go.uber.org/fx v1.20.0
	go.uber.org/zap v1.26.0
	google.golang.org/grpc v1.60.1
	google.golang.org/protobuf v1.32.0
	github.com/golang-migrate/migrate/v4 v4.17.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.0
	github.com/stretchr/testify v1.8.4
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240125205218-1f4bbc51befe // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240125205218-1f4bbc51befe // indirect
)` + "`" + `

// README template
const ReadmeTemplate = ` + "`" + `# {{.ServiceName}}

A gRPC-based microservice built with Go.

## Project Structure

` + "```" + `
{{.ServiceName}}/
├── api/                           # Generated API files
│   ├── contract/proto/            # Protocol buffer definitions
│   └── <service>/                 # Generated protobuf code
├── internal/                      # Private application code
│   ├── models/                    # Data models/entities
│   ├── repository/                # Data access layer
│   ├── service/                   # Business logic
│   ├── handler/                   # gRPC handlers
│   ├── middleware/                # Interceptors and middleware
│   └── config/                    # Configuration
├── server/                        # Server implementations
│   ├── grpc.go                    # gRPC server
│   └── http.go                    # HTTP server
├── pkg/                           # Public packages
│   ├── database/                  # Database utilities
│   └── logger/                    # Logging utilities
├── test/                          # Test infrastructure
│   ├── mocks/                     # Mock implementations
│   └── fixtures/                  # Test fixtures and helpers
├── cmd/                           # Application entry points
├── migrations/                    # Database migrations
├── docker-compose.yml             # Local development setup
├── Dockerfile
└── Makefile
` + "```" + `

## Development

### Prerequisites

- Go 1.21+
- PostgreSQL
- Protocol Buffers compiler (protoc)
- grpc-gateway protoc plugins
- Docker & Docker Compose
- Protocol Buffers compiler (protoc)
- Buf CLI

### Setup

1. Clone the repository
2. Install dependencies:
   ` + "```bash" + `
   go mod download
   ` + "```" + `

3. Start the database:
   ` + "```bash" + `
   docker-compose up -d postgres
   ` + "```" + `

4. Run migrations:
   ` + "```bash" + `
   make migrate-up
   ` + "```" + `

5. Generate protobuf code:
   ` + "```bash" + `
   make proto-gen
   ` + "```" + `

6. Start the server:
   ` + "```bash" + `
   make run
   ` + "```" + `

## Available Commands

- ` + "`make build`" + ` - Build the application
- ` + "`make run`" + ` - Run the application
- ` + "`make test`" + ` - Run tests
- ` + "`make proto-gen`" + ` - Generate protobuf code
- ` + "`make migrate-up`" + ` - Run database migrations
- ` + "`make migrate-down`" + ` - Rollback database migrations
- ` + "`make docker-build`" + ` - Build Docker image
- ` + "`make clean`" + ` - Clean build artifacts

## API Documentation

The service exposes both gRPC and HTTP endpoints via grpc-gateway.

- gRPC: ` + "`localhost:8080`" + `
- HTTP: ` + "`localhost:8081`" + `

## Environment Variables

- ` + "`PORT`" + ` - Server port (default: 8080)
- ` + "`DATABASE_URL`" + ` - PostgreSQL connection string
- ` + "`LOG_LEVEL`" + ` - Log level (debug, info, warn, error)

## Testing

Run unit tests:
` + "```bash" + `
make test
` + "```" + `

Run tests with coverage:
` + "```bash" + `
go test -cover ./...
` + "```" + `

Run integration tests:
` + "```bash" + `
go test -tags=integration ./...
` + "```" + `
` + "`"
