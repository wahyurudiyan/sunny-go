# sgo Makefile

.PHONY: build clean test deps run dev

# Build the sgo CLI
build:
	go build -o bin/sgo ./cmd

# Clean build artifacts
clean:
	rm -rf bin/

# Run the test suite (Ginkgo specs, run via `go test`)
test:
	go test ./...

# Install/tidy dependencies
deps:
	go mod tidy
	go mod download

# Build and run the CLI
run: build
	./bin/sgo

# Run without building a binary first
dev:
	go run ./cmd
