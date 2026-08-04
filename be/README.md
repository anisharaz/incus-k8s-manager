# KOI Backend

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
- Access to an Incus daemon's unix socket (`INCUS_SOCKET_PATH`, see `meta/incusDocker/`)
- (optional) `migrate` CLI (golang-migrate), for manual migration control — the app applies migrations itself on startup

## Setup

1. **Install dependencies:**

   ```bash
   make deps
   ```

2. **Create environment file:**

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

The app applies any pending database migrations itself on every startup
(`db/migrations` is embedded into the binary — see `cmd/server/main.go`'s
`runMigrations`), so there's no separate migration step. The standalone
`migrate` CLI is still useful for manual control (rolling back a migration,
inspecting schema state):

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/incus_k8s_manager?sslmode=disable"
make migrate-down   # roll back the last migration
```

## API Endpoints

> **See [`../API.md`](../API.md) for the full API reference** (request/response
> shapes, error format, the async job-polling pattern, validation rules,
> and a worked example flow) — written for frontend consumption. The
> summary below is just a quick index.

### Health Check

- **GET** `/health` - Server health status

### API v1

- **GET** `/api/v1/` - API root information
- **GET** `/api/v1/status` - Incus service status

### Jobs

- **GET** `/api/v1/jobs` - List all background jobs
- **GET** `/api/v1/jobs/:id` - Get a single job (status, progress, result)

### Authentication

Cookie-based session (JWT in an HttpOnly cookie, see `internal/auth` and
`internal/middleware/auth.go`). Exactly one **admin**, created once via a
bootstrap flow tracked in the `app_states` singleton row; any number of
regular **users**, created by the admin. See `API.md` for the full flow.

- **GET** `/api/v1/auth/status` - `{"adminCreated": bool}` — public
- **POST** `/api/v1/auth/register-admin` - Bootstrap the one admin account (works once), logs it in
- **POST** `/api/v1/auth/login` - `{"username", "password"}` — any role
- **POST** `/api/v1/auth/logout` - Clears the session cookie
- **GET** `/api/v1/auth/me` - Currently authenticated user (session required)

### Users

Admin-only (`401` not logged in, `403` logged in as a non-admin). `ownerId`
on cluster networks/clusters below must reference a real user of either
role (violations surface as a raw FK-violation 500 for now — see `API.md`'s
Known Gaps).

- **POST** `/api/v1/users` - Create a regular user (`{"username", "password"}`, password ≥8 chars)
- **GET** `/api/v1/users` - List all users
- **GET** `/api/v1/users/:id` - Get a single user

### Cluster Networks

An Incus bridge network that cluster VMs are later launched onto. Creation
validates the CIDR against every network Incus currently knows about
(managed or not) to avoid overlaps, then creates the Incus bridge network
synchronously (no background job needed).

- **POST** `/api/v1/networks` - Create a cluster network (`{"ownerId": "...", "name": "...", "cidr": "10.10.0.0/24"}`)
- **GET** `/api/v1/networks` - List all cluster networks
- **GET** `/api/v1/networks/:id` - Get a single cluster network
- **DELETE** `/api/v1/networks/:id` - Delete a cluster network (fails if still in use by an instance)

`name` is a free-form display name, unique per owner. The actual Incus
bridge interface name is system-generated (`incusName` in the response) to
satisfy Incus's restrictive interface naming rules (<=15 characters) without
constraining what the user can call it.

### Clusters

Creating a cluster launches its master node's VM (on the chosen cluster
network) as a background job — see Jobs above to poll progress. For the
master, the job also runs `kubeadm init`, copies `/etc/kubernetes/admin.conf`
to `/root/.kube/config`, and polls the API server's `/healthz` until it's
ready before marking the job complete. No CNI plugin is installed, so
`kubectl get nodes` shows every node as `NotReady` — pod networking is a
later step.

`cpu` (int), `memory`, and `disk` (Incus size format, e.g. `"4GiB"`) size the
node's VM; each is optional and falls back to a default if omitted. All
three are validated against a minimum — `cpu`/`memory` match kubeadm's own
hard preflight requirements (2 CPUs, 1700MB; enforced for `kubeadm join` too,
not just `init`), `disk` is this app's own floor (20GiB, not
kubeadm-enforced) — a value below the minimum is rejected with a 400, not
silently clamped.

- **POST** `/api/v1/clusters` - Create a cluster + master node (`{"ownerId": "...", "networkId": "...", "name": "...", "cpu": 2, "memory": "2GiB", "disk": "20GiB"}`)
- **GET** `/api/v1/clusters` - List all clusters
- **GET** `/api/v1/clusters/:id` - Get a single cluster
- **GET** `/api/v1/clusters/:id/nodes` - List a cluster's nodes (master first), each with its `jobId`, `status`, and `ip`
- **POST** `/api/v1/clusters/:id/nodes` - Add a worker node (body optional — same `cpu`/`memory`/`disk` fields as above, all defaulted if omitted: `{}` works). Requires the cluster to be `ready` and its master `running` (`409` otherwise). Fetches a *fresh* join token from the master (`kubeadm token create --print-join-command` — not the one kubeadm init printed, which may be long expired) and runs `kubeadm join` on the new VM. Display name auto-increments per cluster (`worker-1`, `worker-2`, ...).

Like cluster networks, a node's `name` (e.g. `master`) is a display name
unique within its cluster; `incusName` is the system-generated, globally
unique Incus VM instance name.

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

`internal/jobs/manager.go` holds the generic scaffold: `Manager` plus
`List`/`Get`/`updateJob`. Each job type gets its own file (see
`internal/jobs/node.go`'s `node_provision` type, used by cluster creation to
launch a node's VM without blocking the request):

1. A `Create<Job>Job(...)` method that builds a `models.Job`, persists it as
   `queued`, and starts a goroutine.
2. A `run<Job>Job(...)` method that performs the work and calls `updateJob`
   at each stage to persist `status`/`progress`/`stage`/`message`, updating
   related rows (e.g. the node/cluster) as it goes via direct `m.db` calls.

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

The binary will be created in `bin/koi`.

## Configuration

Environment variables can be set via `.env` file or system environment:

- `PORT` - Server port (default: 8000)
- `ENV` - Environment (development/production, default: development)
- `DATABASE_URL` - Postgres connection string, e.g. `postgres://postgres:postgres@localhost:5432/incus_k8s_manager?sslmode=disable`. Used by both the app and the migrate CLI.
- `DB_HOST` / `DB_PORT` / `DB_NAME` / `DB_USER` / `DB_PASSWORD` - Fallback DB settings used to build the DSN when `DATABASE_URL` is not set.
- `INCUS_SOCKET_PATH` - Path to the Incus unix socket (default: `/shared-socket/incus.sock`). Empty falls back to `$INCUS_SOCKET`/`$INCUS_DIR`/standard system paths.
- `JWT_SECRET` - Signing key for session tokens. If unset, a random one is generated at startup (logged as a warning) — fine for a dev box, but every restart invalidates all sessions. Set this in any deployment that should survive a restart.

## Dependencies

- `github.com/gofiber/fiber/v3` - Web framework
- `github.com/gofiber/schema` - Request validation
- `github.com/google/uuid` - UUID generation
- `github.com/lxc/incus/v7` - Incus Go SDK (instance lifecycle, exec, networks)
- `github.com/golang-jwt/jwt/v5` - Session tokens
- `golang.org/x/crypto/bcrypt` - Password hashing

## Notes

- The API is configured to allow requests from `localhost:5173` and `localhost:8000`, with credentials (cookies) enabled
- Modify CORS settings in `internal/middleware/cors.go` as needed
- `/api/v1/status` checks Incus reachability via the SDK over the shared socket (`internal/incus.Client.List`) — no `incus` CLI binary needed

## Future Enhancements

- [ ] Recovery of interrupted jobs on server restart (currently marked by absence; orphaned `running` jobs are not re-run)
- [ ] CNI installation (nodes stay `NotReady` in `kubectl get nodes` without one)
- [ ] Deleting a cluster/node (stop + delete the Incus VM, not just the DB row)
- [ ] Removing a worker cleanly (`kubectl drain` + `kubeadm reset` before deleting its VM, not just deleting the VM out from under the cluster)
- [ ] Authentication and authorization
- [ ] Ownership existence checks (an invalid `ownerId`/`networkId` currently surfaces as a raw FK-violation 500, not a clean 400/404)
- [ ] Comprehensive error handling
- [ ] Request validation middleware
- [ ] API documentation (Swagger)
- [ ] Unit tests
- [ ] Integration tests
