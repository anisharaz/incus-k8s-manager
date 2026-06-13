.PHONY: build run dev test clean fmt lint help

# Variables
BINARY_NAME=incus-k8s-manager
MAIN_PATH=./cmd/server/main.go

help:
	@echo "Available commands:"
	@echo "  make build       - Build the application"
	@echo "  make run         - Run the application"
	@echo "  make dev         - Run in development mode with hot reload (requires air)"
	@echo "  make test        - Run tests"
	@echo "  make fmt         - Format code"
	@echo "  make lint        - Run linter"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make deps        - Download dependencies"

build:
	@echo "Building..."
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: bin/$(BINARY_NAME)"

run: build
	@echo "Running..."
	./bin/$(BINARY_NAME)

dev:
	@echo "Running in development mode with hot reload..."
	air

test:
	@echo "Running tests..."
	go test -v ./...

fmt:
	@echo "Formatting code..."
	go fmt ./...

lint:
	@echo "Running linter..."
	golangci-lint run ./...

clean:
	@echo "Cleaning..."
	rm -rf bin/
	go clean

deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies updated"
