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
│   │   ├── networks.go      # Cluster network HTTP handlers
│   │   ├── health.go        # Health/status handlers
│   │   └── jobs.go          # Job HTTP handlers (list/get)
│   ├── incus/
│   │   ├── client.go        # Incus SDK connection + instance lifecycle
│   │   ├── exec.go          # In-VM command execution
│   │   └── network.go       # Incus network management
│   ├── jobs/
│   │   └── manager.go       # DB-backed background job manager (scaffold, no job types yet)
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
- **DB-Backed Job Manager**: Long-running tasks (e.g. VM provisioning) run in background goroutines and persist their status/progress to Postgres continuously, so they survive restarts and can be polled via the API.
- **Incus Integration**: Cluster networks and (soon) VM lifecycle are managed via the Incus Go SDK over a shared unix socket.
- **Schema Migrations**: Database schema managed with the `golang-migrate` CLI.
- **CORS Support**: Configured for frontend development
- **Middleware**: Request logging and error handling
- **Environment Configuration**: Configurable via environment variables

## Prerequisites

- Go 1.26 or higher
- PostgreSQL
- `migrate` CLI (golang-migrate) for database migrations
- Incus CLI installed (for status checks)
- Access to an Incus daemon's unix socket (`INCUS_SOCKET_PATH`, see `meta/incusDocker/`)

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

### Cluster Networks

An Incus bridge network that cluster VMs are later launched onto. Creation
validates the CIDR against every network Incus currently knows about
(managed or not) to avoid overlaps, then creates the Incus bridge network
synchronously (no background job needed).

- **POST** `/api/v1/networks` - Create a cluster network (`{"name": "...", "cidr": "10.10.0.0/24"}`)
- **GET** `/api/v1/networks` - List all cluster networks
- **GET** `/api/v1/networks/:id` - Get a single cluster network
- **DELETE** `/api/v1/networks/:id` - Delete a cluster network (fails if still in use by an instance)

`name` doubles as the underlying Linux bridge interface name, so it must
satisfy Incus's interface naming rules (2-15 characters, `[-_a-zA-Z0-9.]`).

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

The job manager (`internal/jobs/manager.go`) is intentionally a bare
scaffold: `Manager` plus `List`/`Get`/`updateJob`. It has no job types yet
(cluster network creation is synchronous and doesn't need one). To add one
for a long-running task (e.g. VM provisioning):

1. Add a `Create<Job>Job(...)` method that builds a `models.Job`, persists it
   as `queued`, and starts a goroutine.
2. Add a `run<Job>Job(...)` method that performs the work and calls
   `updateJob` at each stage to persist `status`/`progress`/`stage`/`message`.

## Incus Client

`internal/incus` wraps the Incus Go SDK (`github.com/lxc/incus/v7/client`)
over the unix socket shared by the incus container (see
`meta/incusDocker/docker-compose.yml`'s `incus-socket-share` volume, mounted
read-only into this app's container as `/shared-socket/incus.sock`, the
`INCUS_SOCKET_PATH` default). It covers instance lifecycle (`Launch`,
`Start`, `Stop`, `Delete`, `Get`, `List`, `WaitForIPv4`, `WaitForAgent`),
in-VM command execution (`Exec`), and network management (`CreateNetwork`,
`ListNetworks`, `GetNetwork`, `DeleteNetwork`).

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
- `INCUS_SOCKET_PATH` - Path to the Incus unix socket (default: `/shared-socket/incus.sock`). Empty falls back to `$INCUS_SOCKET`/`$INCUS_DIR`/standard system paths.

## Dependencies

- `github.com/gofiber/fiber/v3` - Web framework
- `github.com/gofiber/schema` - Request validation
- `github.com/google/uuid` - UUID generation
- `github.com/lxc/incus/v7` - Incus Go SDK (instance lifecycle, exec, networks)

## Notes

- The API is configured to allow requests from `localhost:5173` and `localhost:8000`
- Modify CORS settings in `internal/middleware/cors.go` as needed
- Incus status checks require the Incus CLI to be installed and accessible

## Future Enhancements

- [ ] Recovery of interrupted jobs on server restart (currently marked by absence; orphaned `running` jobs are not re-run)
- [ ] VM lifecycle jobs (launch nodes onto a cluster network, run `kubeadm`/join via `incus.Exec`)
- [ ] Authentication and authorization
- [ ] Kubernetes cluster integration
- [ ] Comprehensive error handling
- [ ] Request validation middleware
- [ ] API documentation (Swagger)
- [ ] Unit tests
- [ ] Integration tests
