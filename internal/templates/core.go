package templates

// MainTemplate defines the main.go file template
const MainTemplate = `package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "{{.ProjectName}}",
	})

	// Middleware
	app.Use(cors.New())
	app.Use(logger.New())

	// Routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Welcome to {{.ProjectName}}!",
			"status":  "healthy",
		})
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
		})
	})

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Start server
	log.Printf("🚀 Server starting on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
`

// GoModTemplate defines the go.mod file template
const GoModTemplate = `module {{.ProjectName}}

go 1.21

require (
	github.com/gofiber/fiber/v2 v2.52.0
)
`

// ReadmeTemplate defines the README.md file template
const ReadmeTemplate = `# {{.ProjectName}}

A modern Go web service built with Fiber framework.

## Quick Start

### Prerequisites
- Go 1.21 or higher

### Installation
` + "`" + `bash
# Clone the repository
git clone <your-repo-url>
cd {{.ProjectName}}

# Install dependencies
go mod tidy

# Run the application
go run main.go
` + "`" + `

### Using Makefile
` + "`" + `bash
# Build the application
make build

# Run the application
make run

# Clean build artifacts
make clean
` + "`" + `

## API Endpoints

- ` + "`" + `GET /` + "`" + ` - Welcome message
- ` + "`" + `GET /health` + "`" + ` - Health check

## Environment Variables

- ` + "`" + `PORT` + "`" + ` - Server port (default: 3000)

## Development

The application uses:
- [Fiber](https://gofiber.io/) - Web framework
- Built-in middleware for CORS and logging

## Building

` + "`" + `bash
go build -o bin/{{.ProjectName}} main.go
` + "`" + `

## License

MIT License
`

// MakefileTemplate defines the Makefile template
const MakefileTemplate = `.PHONY: build run clean test

# Build the application
build:
	go build -o bin/{{.ProjectName}} ./cmd/{{.ProjectName}}

# Run the application
run:
	go run ./cmd/{{.ProjectName}}

# Clean build artifacts
clean:
	rm -rf bin/

# Run tests
test:
	go test -v ./...

# Install dependencies
deps:
	go mod tidy

# Development mode with hot reload (requires air)
dev:
	air

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run
`

// GitignoreTemplate defines the .gitignore file template
const GitignoreTemplate = `# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib
bin/

# Test binary, built with go test -c
*.test

# Output of the go coverage tool
*.out

# Dependency directories
vendor/

# Go workspace file
go.work

# IDE files
.vscode/
.idea/
*.swp
*.swo

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db

# Environment files
.env
.env.local
.env.production

# Logs
*.log
logs/

# Runtime data
pids
*.pid
*.seed
*.pid.lock

# Coverage directory used by tools like istanbul
coverage/

# Air (live reload) temporary files
tmp/
`
