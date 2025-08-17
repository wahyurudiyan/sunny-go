package templates

// Docker and deployment templates

// Makefile template
const MakefileTemplate = `APP_NAME={{.ServiceName}}
DOCKER_IMAGE={{.ServiceName}}:latest

.PHONY: build run test clean proto-gen migrate-up migrate-down docker-build

# Build the application
build:
	go build -o bin/$(APP_NAME) ./cmd

# Run the application
run: build
	./bin/$(APP_NAME)

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -cover ./...

# Run integration tests
test-integration:
	go test -tags=integration -v ./...

# Generate protobuf code
proto-gen:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative \
		--grpc-gateway_opt=generate_unbound_methods=true \
		api/contract/proto/*.proto

# Generate code for specific service
proto-gen-service:
	@if [ -z "$(SERVICE)" ]; then echo "Usage: make proto-gen-service SERVICE=<service_name>"; exit 1; fi
	protoc --go_out=api/$(SERVICE) --go_opt=paths=source_relative \
		--go-grpc_out=api/$(SERVICE) --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=api/$(SERVICE) --grpc-gateway_opt=paths=source_relative \
		--grpc-gateway_opt=generate_unbound_methods=true \
		api/contract/proto/$(SERVICE).proto

# Run database migrations up
migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

# Run database migrations down
migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down

# Build Docker image
docker-build:
	docker build -t $(DOCKER_IMAGE) .

# Clean build artifacts
clean:
	rm -rf bin/
	find api -name "*.pb.go" -delete

# Install development dependencies
dev-deps:
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/bufbuild/buf/cmd/buf@latest

# Start development environment
dev-up:
	docker-compose up -d

# Stop development environment
dev-down:
	docker-compose down

# View logs
logs:
	docker-compose logs -f

# Reset database
db-reset: migrate-down migrate-up

# Lint code
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Tidy dependencies
tidy:
	go mod tidy
`

// Dockerfile template
const DockerfileTemplate = `FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd

# Final stage
FROM alpine:latest

WORKDIR /root/

# Install ca-certificates
RUN apk --no-cache add ca-certificates

# Copy the binary
COPY --from=builder /app/main .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
`

// Docker Compose template
const DockerComposeTemplate = `version: '3.8'

services:
  {{.ServiceName}}:
    build: .
    ports:
      - "8080:8080"
      - "8081:8081"
    environment:
      - PORT=8080
      - DATABASE_URL=postgres://user:password@postgres:5432/{{.ServiceName}}_db?sslmode=disable
      - LOG_LEVEL=info
    depends_on:
      - postgres
    networks:
      - {{.ServiceName}}_network

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB={{.ServiceName}}_db
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    networks:
      - {{.ServiceName}}_network

  migrate:
    image: migrate/migrate
    networks:
      - {{.ServiceName}}_network
    volumes:
      - ./migrations:/migrations
    command: [
      "-path", "/migrations",
      "-database", "postgres://user:password@postgres:5432/{{.ServiceName}}_db?sslmode=disable",
      "up"
    ]
    depends_on:
      - postgres

volumes:
  postgres_data:

networks:
  {{.ServiceName}}_network:
    driver: bridge
`
