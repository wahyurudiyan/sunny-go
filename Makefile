# test-setup Makefile

.PHONY: build clean proto test

# Build the application
build:
	go build -o bin/sunny ./cmd

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf generated/

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@for proto in $$(find proto -name "*.proto"); do \
		echo "Processing $$proto"; \
		protoc --proto_path=proto \
			--go_out=generated \
			--go_opt=paths=source_relative \
			--go-grpc_out=generated \
			--go-grpc_opt=paths=source_relative \
			--go-http_out=generated \
			--go-http_opt=paths=source_relative \
			$$proto; \
	done

# Run tests
test:
	go test ./...

# Install dependencies
deps:
	go mod tidy
	go mod download

# Run the application
run: build
	./bin/test-setup

# Development server with hot reload
dev:
	go run ./cmd/test-setup
