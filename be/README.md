# Incus K8s Manager - Backend

A Fiber-based REST API for managing Incus containers and Kubernetes integration, with a database-backed background job manager.

## Project Structure

```
be/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── db/
│   └── migrations/           # golang-migrate SQL migrations
├── internal/
│   ├── config/
│   │   └── config.go        # Configuration management
│   ├── handlers/
│   │   ├── clusters.go      # Cluster HTTP handlers
│   │   ├── health.go        # Health/status handlers
│   │   └── jobs.go          # Job HTTP handlers (list/get)
│   ├── jobs/
│   │   └── manager.go       # DB-backed background job manager
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
├── Makefile                  # Build, run, and migrate scripts
└── README.md                 # This file
```

## Features

- **Fiber Framework**: Fast and lightweight web framework
- **DB-Backed Job Manager**: Long-running tasks (e.g. cluster creation) run in background goroutines and persist their status/progress to Postgres continuously, so they survive restarts and can be polled via the API.
- **Schema Migrations**: Database schema managed with the `golang-migrate` CLI.
- **CORS Support**: Configured for frontend development
- **Middleware**: Request logging and error handling
- **Environment Configuration**: Configurable via environment variables

## Prerequisites

- Go 1.26 or higher
- PostgreSQL
- `migrate` CLI (golang-migrate) for database migrations
- Incus CLI installed (for status checks)

## Setup

1. **Install dependencies:**

   ```bash
   make deps
   ```

2. **Install the migrate CLI (if not already installed):**

   ```bash
   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
   ```

3. **Create environment file:**

   ```bash
   cp .env.example .env
   ```

4. **Apply database migrations:**

   ```bash
   export DATABASE_URL="postgres://postgres:postgres@localhost:5432/incus_k8s_manager?sslmode=disable"
   make migrate-up
   ```

5. **Run in development mode (with hot reload):**

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

### Jobs

- **GET** `/api/v1/jobs` - List all background jobs
- **GET** `/api/v1/jobs/:id` - Get a single job (status, progress, result)

### Clusters

- **POST** `/api/v1/clusters` - Create a cluster (starts a background job)
- **GET** `/api/v1/clusters` - List all clusters
- **GET** `/api/v1/clusters/:id` - Get a single cluster

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
make migrate-up  # Apply pending migrations (requires DATABASE_URL)
make migrate-down # Roll back the last migration (requires DATABASE_URL)
make migrate-create NAME=create_xyz # Create a new migration pair
make help        # Show this help message
```

## Adding a New Job Type

The job manager is intentionally simple. Each job is a pair of methods on
`internal/jobs/manager.go`:

1. A `Create<Job>Job(...)` method that builds a `models.Job`, persists it as
   `queued`, and starts a goroutine.
2. A `run<Job>Job(...)` method that performs the work and calls `updateJob`
   at each stage to persist `status`/`progress`/`stage`/`message`, updating
   related records (e.g. the cluster row) as needed.

`runClusterJob` uses a `provisioningStep` slice to run a series of commands
(currently sample `sleep` placeholders) and parses `Key: Value` lines from
their combined output via `extractDetails` (so an `IP:` line lands in the job
result and the cluster's `ip` column). Replace the sample steps with your real
provisioning commands (e.g. `incus launch`, `incus exec`).

See `CreateClusterJob`/`runClusterJob` for the reference pattern.

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
- `DATABASE_URL` - Postgres connection string, e.g. `postgres://postgres:postgres@localhost:5432/incus_k8s_manager?sslmode=disable`. Used by both the app and the migrate CLI.
- `DB_HOST` / `DB_PORT` / `DB_NAME` / `DB_USER` / `DB_PASSWORD` - Fallback DB settings used to build the DSN when `DATABASE_URL` is not set.

## Dependencies

- `github.com/gofiber/fiber/v3` - Web framework
- `github.com/gofiber/schema` - Request validation
- `github.com/google/uuid` - UUID generation

## Notes

- The API is configured to allow requests from `localhost:5173` and `localhost:8000`
- Modify CORS settings in `internal/middleware/cors.go` as needed
- Incus status checks require the Incus CLI to be installed and accessible

## Future Enhancements

- [ ] Recovery of interrupted jobs on server restart (currently marked by absence; orphaned `running` jobs are not re-run)
- [ ] Incus VM creation / setup jobs (follow the `CreateClusterJob` pattern)
- [ ] Authentication and authorization
- [ ] Container lifecycle management endpoints
- [ ] Kubernetes cluster integration
- [ ] Comprehensive error handling
- [ ] Request validation middleware
- [ ] API documentation (Swagger)
- [ ] Unit tests
- [ ] Integration tests
