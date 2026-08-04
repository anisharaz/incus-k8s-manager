.PHONY: build run dev test clean fmt lint help migrate-up migrate-down migrate-create

# Variables
BINARY_NAME=koi
MAIN_PATH=./cmd/server/main.go
MIGRATE_PATH=db/migrations

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
	@echo "  make migrate-up  - Apply all pending migrations (requires DATABASE_URL)"
	@echo "  make migrate-down - Roll back the last migration (requires DATABASE_URL)"
	@echo "  make migrate-create NAME=create_xyz - Create a new migration pair"

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

migrate-up:
	@test -n "$${DATABASE_URL}" || (echo "DATABASE_URL is required (e.g. postgres://user:pass@localhost:5432/dbname?sslmode=disable)"; exit 1)
	@echo "Applying migrations..."
	migrate -path $(MIGRATE_PATH) -database "$${DATABASE_URL}" up

migrate-down:
	@test -n "$${DATABASE_URL}" || (echo "DATABASE_URL is required (e.g. postgres://user:pass@localhost:5432/dbname?sslmode=disable)"; exit 1)
	@echo "Rolling back the last migration..."
	migrate -path $(MIGRATE_PATH) -database "$${DATABASE_URL}" down 1

migrate-create:
	@test -n "$(NAME)" || (echo "Usage: make migrate-create NAME=create_something"; exit 1)
	migrate create -ext sql -dir $(MIGRATE_PATH) -seq $(NAME)
