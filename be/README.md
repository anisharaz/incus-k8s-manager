# Incus K8s Manager - Backend

A Fiber-based REST API for managing Incus containers and Kubernetes integration.

## Project Structure

```
be/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go        # Configuration management
│   ├── handlers/
│   │   └── health.go        # HTTP handlers
│   ├── middleware/
│   │   ├── cors.go          # CORS middleware
│   │   └── logger.go        # Request logging middleware
│   ├── models/
│   │   └── models.go        # Data models and structs
│   └── routes/
│       └── routes.go        # Route definitions
├── go.mod                    # Go module definition
├── go.sum                    # Dependency checksums
├── .env.example              # Environment variables template
├── Makefile                  # Build and run scripts
└── README.md                 # This file
```

## Features

- **Fiber Framework**: Fast and lightweight web framework
- **Modular Structure**: Organized packages for handlers, middleware, models, and configuration
- **CORS Support**: Configured for frontend development
- **Middleware**: Request logging and error handling
- **Environment Configuration**: Configurable via environment variables

## Prerequisites

- Go 1.26 or higher
- Incus CLI installed (for status checks)

## Setup

1. **Install dependencies:**

   ```bash
   make deps
   ```

2. **Create environment file (optional):**

   ```bash
   cp .env.example .env
   ```

3. **Run in development mode (with hot reload):**

   ```bash
   make dev
   ```

   Or build and run:

   ```bash
   make run
   ```

## API Endpoints

### Health Check

- **GET** `/health` - Server health status

### API v1

- **GET** `/api/v1/` - API root information
- **GET** `/api/v1/status` - Incus service status

## Available Commands

```bash
make build       # Build the application
make run         # Build and run
make dev         # Run with hot reload (requires air)
make test        # Run tests
make fmt         # Format code
make lint        # Run linter
make clean       # Clean build artifacts
make deps        # Download dependencies
make help        # Show this help message
```

## Development

The project uses `air` for hot reloading during development. Configuration is in `.air.toml`.

To start development:

```bash
make dev
```

This will automatically rebuild and restart the server when you save files.

## Building for Production

```bash
make build
```

The binary will be created in `bin/incus-k8s-manager`.

## Configuration

Environment variables can be set via `.env` file or system environment:

- `PORT` - Server port (default: 8000)
- `ENV` - Environment (development/production, default: development)

## Dependencies

- `github.com/gofiber/fiber/v3` - Web framework
- `github.com/gofiber/schema` - Request validation
- `github.com/google/uuid` - UUID generation

## Notes

- The API is configured to allow requests from `localhost:5173` and `localhost:8000`
- Modify CORS settings in `internal/middleware/cors.go` as needed
- Incus status checks require the Incus CLI to be installed and accessible

## Future Enhancements

- [ ] Database integration (PostgreSQL/MySQL)
- [ ] Authentication and authorization
- [ ] Container lifecycle management endpoints
- [ ] Kubernetes cluster integration
- [ ] Comprehensive error handling
- [ ] Request validation middleware
- [ ] API documentation (Swagger)
- [ ] Unit tests
- [ ] Integration tests
