APP_NAME=test-service
DOCKER_IMAGE=test-service:latest

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
