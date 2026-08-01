# API Reference

REST API for the Incus K8s Manager backend (`be/`). This is the contract the
frontend (`fe/`) codes against.

- **Base URL (dev):** `http://localhost:8000` (`PORT` env var, see `be/.env.example`)
- **Format:** all requests/responses are JSON (`Content-Type: application/json`)
- **Auth:** none yet. Every resource is tied to a `User` (`ownerId`) purely
  for ownership bookkeeping — there is no login, no session, no token. Any
  client can act as any user by passing their `id`.
- **CORS (dev):** `http://localhost:5173`, `http://localhost:8000`,
  `http://localhost:3000` are allowed (`be/internal/middleware/cors.go`).
  Vite's default port (5173) already works out of the box.

---

## Conventions

### IDs

Every resource `id` is a server-generated UUID v4 string. Always sent by
the server, never chosen by the client.

### Timestamps

`createdAt`, `updatedAt`, `completedAt` are RFC 3339 timestamps with
nanosecond precision and a timezone offset, e.g.
`"2026-08-02T02:56:56.61614602+05:30"`. Treat as opaque strings and parse
with a standard date library — don't assume UTC or a fixed fractional-second
length.

### Two names on Networks and Nodes

`ClusterNetwork` and `Node` each carry **two** name fields:

- `name` — the display name you chose (or the server auto-generated, e.g.
  `worker-1`). Free-form, unique only within its own scope (see each
  resource below).
- `incusName` — a server-generated identifier used as the actual Incus
  resource name (bridge interface / VM instance name) and globally unique
  across the whole system. You'll rarely need it in the UI, but it's
  included since it's occasionally useful for debugging (it's what shows up
  if you ever shell into the Incus host).

**Never construct `incusName` yourself or use it as a form input** — it's
generated server-side and returned in every response for the resource.

### Errors

Every non-2xx response has this exact shape:

```json
{
  "error": "validation error",
  "message": "cpu must be at least 2, got 1",
  "code": 400
}
```

- `error` — short machine-ish category (e.g. `"validation error"`,
  `"not found"`, `"database error"`, `"incus error"`). Not a stable enum —
  treat it as a label for logging/display, not something to switch on.
  Switch on the HTTP status code instead.
- `message` — human-readable detail, safe to show directly in a UI toast/
  form error.
- `code` — repeats the HTTP status code.

Status codes used throughout the API:

| Code | Meaning here |
|---|---|
| `200` | Success (GET) |
| `201` | Created (resource fully created synchronously — Users, Cluster Networks) |
| `202` | Accepted (resource row created, but a background job is still provisioning it — Clusters, Nodes) |
| `204` | Success, no body (DELETE) |
| `400` | Bad request body / validation failure — safe to show `message` next to the offending field |
| `404` | Resource not found |
| `409` | Conflict — duplicate name, CIDR overlap, or an operation attempted in the wrong state (e.g. adding a worker before the master is ready) |
| `500` | Server/database/Incus error. **Known gap:** an invalid `ownerId` or `networkId` that doesn't reference a real row currently surfaces here as a raw Postgres foreign-key-violation message, not a clean `400`/`404`. Validate that IDs came from a real prior API response before submitting. |

### The async job pattern (important for the UI)

Creating a **Cluster** or a **Node** does real work that takes anywhere from
~10 seconds (a worker joining) to several minutes (a master's first
`kubeadm init`, which pulls container images). These endpoints return
**`202 Accepted`** immediately with the resource in a non-final state, and
kick off a background **Job**. The UI must poll to find out when it's done:

1. `POST /api/v1/clusters` → get back a `Cluster` (`status: "creating"`).
2. `GET /api/v1/clusters/:id/nodes` → find the node (there's exactly one:
   the master), read its `jobId`.
3. Poll `GET /api/v1/jobs/:id` (e.g. every 3–5s) and show `stage` /
   `progress` / `message` as a progress indicator.
4. When the job's `status` becomes `"succeeded"` or `"failed"`, stop
   polling. Re-fetch the `Cluster`/`Node` — their `status`/`message` will
   reflect the outcome too (`ready`/`running` or `failed`).

The same pattern applies to `POST /api/v1/clusters/:id/nodes` (adding a
worker) — poll the returned `Node`'s `jobId`.

**Job `stage` values you'll see** (informational strings for progress UI,
not a fixed enum — don't hardcode exhaustive handling, just display them):

| Stage | Meaning | Applies to |
|---|---|---|
| `queued` | Job row created, goroutine not yet scheduled | both |
| `launching` | Creating and starting the Incus VM | both |
| `waiting-for-ip` | Waiting for DHCP to assign the VM an address | both |
| `waiting-for-agent` | Waiting for the in-VM guest agent to respond | both |
| `waiting-for-containerd` | Waiting for the container runtime to finish starting | both |
| `bootstrapping` | Running `kubeadm init` | master only |
| `configuring-kubeconfig` | Copying `admin.conf` for `kubectl` | master only |
| `joining` | Fetching a join token from the master and running `kubeadm join` | worker only |
| `verifying` | Polling the API server / node registration before declaring done | both |
| `complete` | Done — check `status` for success/failure | both |
| `failed` | Something errored — see the job's `error` field | both |

---

## Data model

```
User
 ├─ owns → ClusterNetwork (many)
 └─ owns → Cluster (many)
              ├─ references → ClusterNetwork (one; RESTRICT delete while referenced)
              └─ has → Node (many: exactly one "master", zero+ "worker")
                          └─ tracked by → Job (the node's provisioning job)
```

- A `Cluster` always has **exactly one** master node (enforced server-side —
  a cluster is created together with its master in one request).
- A `Cluster`'s own `status` reflects its **master's** provisioning outcome
  (`ready` once the master's `kubeadm init` succeeds and the API server is
  healthy). Worker outcomes don't change cluster status.
- Deleting a `ClusterNetwork` is blocked (`500`, underlying FK violation)
  while any `Cluster` references it.

### Enums

```ts
ClusterNetworkStatus = "creating" | "ready" | "failed"
ClusterStatus        = "creating" | "ready" | "failed" | "deleting"  // "deleting" is defined but not yet used by any endpoint
NodeRole              = "master" | "worker"
NodeStatus            = "creating" | "running" | "stopped" | "failed" | "deleting"  // "stopped"/"deleting" defined but not yet used
JobStatus             = "queued" | "running" | "succeeded" | "failed"
```

---

## Health & Status

### `GET /health`

Liveness check, no `/api/v1` prefix.

```json
{ "status": "ok", "message": "Server is running" }
```

### `GET /api/v1/status`

Whether the Incus CLI is reachable from the backend process. Mostly a
backend-ops diagnostic, not something to build UI around.

```json
{ "status": { "incus": "running" } }
```

`incus` is one of `"running"`, `"stopped"`, `"not found"`.

---

## Users

No auth — see Conventions above. `ownerId` on Networks/Clusters must be a
real user's `id`.

### `POST /api/v1/users`

**Request:**
```json
{ "username": "alice" }
```

**Response `201`:**
```json
{
  "user": {
    "id": "2b9dc998-2c29-4aef-90af-58a938a3d013",
    "username": "alice",
    "createdAt": "2026-08-02T01:28:58.841899082+05:30",
    "updatedAt": "2026-08-02T01:28:58.841899082+05:30"
  }
}
```

**Errors:** `400` empty/>63-char username · `409` username already taken.

### `GET /api/v1/users`

**Response `200`:** `{ "users": [ User, ... ] }` (newest first)

### `GET /api/v1/users/:id`

**Response `200`:** `{ "user": User }` · **`404`** if not found.

> There is currently no `DELETE /api/v1/users/:id` — deliberately removed.
> Deleting users (and cascading to their owned resources, including live
> Incus VMs/networks) is unimplemented; don't build a "delete user" UI yet.

---

## Cluster Networks

An Incus bridge network that cluster VMs are later launched onto. Must be
created before a cluster can be created.

### `POST /api/v1/networks`

Validates the CIDR against **every** network Incus currently knows about
(not just ones created through this API — including the appliance's own
bridge) to prevent an overlapping subnet, then creates the Incus bridge
synchronously (this one's `201`, not `202` — no job/polling needed).

**Request:**
```json
{
  "ownerId": "2b9dc998-2c29-4aef-90af-58a938a3d013",
  "name": "prod-net",
  "cidr": "10.10.0.0/24"
}
```

- `name` — free-form, 1–63 chars, unique **per owner** (two different users
  can both have a network named `"prod-net"`).
- `cidr` — IPv4, must be the network address itself (no host bits — e.g.
  `10.10.0.5/24` is rejected with a suggestion), prefix length between `/8`
  and `/29`.

**Response `201`:**
```json
{
  "network": {
    "id": "5c701cdc-496d-42aa-802c-fa065e2a83a0",
    "ownerId": "2b9dc998-2c29-4aef-90af-58a938a3d013",
    "name": "prod-net",
    "incusName": "cn5c701cdc496d4",
    "cidr": "10.10.0.0/24",
    "gateway": "10.10.0.1",
    "status": "ready",
    "message": "Network created",
    "createdAt": "2026-08-02T01:29:09.073878164+05:30",
    "updatedAt": "2026-08-02T01:29:09.073878164+05:30"
  }
}
```

`gateway` is auto-derived as the first usable address in the CIDR (network
address + 1) — not user-supplied.

**Errors:**
- `400` — missing `ownerId`, bad `name` length, malformed/out-of-range `cidr`
- `409` `"network already exists"` — this owner already has a network with that `name`
- `409` `"cidr conflict"` — overlaps an existing Incus network; `message` names which one and its CIDR
- `500` — bad `ownerId` (FK violation, see Conventions), or an Incus-side error

### `GET /api/v1/networks`

**Response `200`:** `{ "networks": [ ClusterNetwork, ... ] }` (newest first, **all owners** — there's no `?ownerId=` filter yet, filter client-side if needed)

### `GET /api/v1/networks/:id`

**Response `200`:** `{ "network": ClusterNetwork }` · **`404`** if not found.

### `DELETE /api/v1/networks/:id`

Deletes from both Incus and the database. **`204`** on success.

**Errors:** `404` not found · `500` if Incus refuses (e.g. still referenced
by a `Cluster` — message will mention it's in use).

---

## Clusters

Creating a cluster creates its **master node** and launches that node's VM
as a background job (see "The async job pattern" above) — **poll before
assuming the cluster is usable.**

### `POST /api/v1/clusters`

**Request:**
```json
{
  "ownerId": "2b9dc998-2c29-4aef-90af-58a938a3d013",
  "networkId": "5c701cdc-496d-42aa-802c-fa065e2a83a0",
  "name": "prod-cluster",
  "cpu": 2,
  "memory": "2GiB",
  "disk": "20GiB"
}
```

- `name` — free-form, 1–63 chars, unique **per owner**.
- `cpu`, `memory`, `disk` — **all optional**, size the master's VM. Omit any
  of them (or send `0`/`""`) to use the default. If provided, each is
  checked against a minimum and the request is **rejected with `400`** if
  below it — nothing is silently clamped up.

  | Field | Type | Default | Minimum | Why |
  |---|---|---|---|---|
  | `cpu` | int (vCPUs) | `2` | `2` | kubeadm's own hard preflight check |
  | `memory` | string, [Incus size format](#size-string-format) | `"2GiB"` | `1700MB` | kubeadm's own hard preflight check (default is intentionally above the bare minimum — see note below) |
  | `disk` | string, [Incus size format](#size-string-format) | `"20GiB"` | `20GiB` | not kubeadm-enforced; this app's own floor for etcd + images |

  Note: the memory *default* (`2GiB`) is deliberately above the *minimum*
  (`1700MB`) — virtualization overhead can make the guest see slightly less
  RAM than configured, and kubeadm's check is a hard cutoff, so sitting
  exactly on the minimum risks failing it. If a user explicitly requests
  something between `1700MB` and `2GiB`, that's allowed (it passed the
  minimum); the risk only applies to the auto-default.

**Response `202`:**
```json
{
  "cluster": {
    "id": "cba22032-aca2-4d1e-902f-289459c91961",
    "ownerId": "2b9dc998-2c29-4aef-90af-58a938a3d013",
    "networkId": "5c701cdc-496d-42aa-802c-fa065e2a83a0",
    "name": "prod-cluster",
    "status": "creating",
    "message": "Cluster creation started",
    "createdAt": "2026-08-02T01:43:41.23486713+05:30",
    "updatedAt": "2026-08-02T01:43:41.23486713+05:30"
  }
}
```

Immediately follow up with `GET /api/v1/clusters/:id/nodes` to get the
master node's `jobId` to poll (see below) — the create response itself
doesn't include the node or job.

When the master's job succeeds, a follow-up `GET /api/v1/clusters/:id` will
show:
```json
{ "status": "ready", "message": "Kubernetes control plane is ready", ... }
```
or on failure:
```json
{ "status": "failed", "message": "Master node provisioning failed", ... }
```
(the underlying error detail is on the **job**, not the cluster — see Jobs).

**Errors:**
- `400` — missing `ownerId`/`networkId`, bad `name`, or `cpu`/`memory`/`disk` below minimum (message names which field and by how much)
- `404` `"cluster network not found"` — bad `networkId`
- `409` `"cluster already exists"` — this owner already has a cluster with that `name`
- `500` — bad `ownerId` (FK violation), or a job-creation/database error

### `GET /api/v1/clusters`

**Response `200`:** `{ "clusters": [ Cluster, ... ] }` (newest first, all owners)

### `GET /api/v1/clusters/:id`

**Response `200`:** `{ "cluster": Cluster }` · **`404`** if not found.

### `GET /api/v1/clusters/:id/nodes`

Lists the cluster's nodes — master first, then workers in the order they
were added. This is how the UI discovers node IDs, `jobId`s, IPs, and
per-node status.

**Response `200`:**
```json
{
  "nodes": [
    {
      "id": "0f9bf446-fb31-474b-9edb-71fb8f29dfd9",
      "clusterId": "c0150a11-3a9b-4a96-8182-aacae98c33fd",
      "jobId": "cdd9ef13-b5c5-426f-9d1e-d3e07b69c81b",
      "name": "master",
      "incusName": "master-0f9bf446fb31",
      "role": "master",
      "status": "running",
      "ip": "10.44.0.192",
      "message": "Kubernetes control plane is ready",
      "createdAt": "2026-08-02T03:08:02.276843+05:30",
      "updatedAt": "2026-08-02T03:09:14.547593+05:30"
    },
    {
      "id": "d6da0b00-9b36-47e5-89e2-7a3904199423",
      "clusterId": "c0150a11-3a9b-4a96-8182-aacae98c33fd",
      "jobId": "a0ed32d1-408a-4376-bd8e-5d54e9126bff",
      "name": "worker-1",
      "incusName": "worker-d6da0b009b36",
      "role": "worker",
      "status": "running",
      "ip": "10.44.0.9",
      "message": "Node joined the cluster",
      "createdAt": "2026-08-02T03:09:28.180597+05:30",
      "updatedAt": "2026-08-02T03:10:06.962443+05:30"
    }
  ]
}
```

`jobId` is present once the node's provisioning job has been created
(effectively always, immediately after the node row exists) — treat it as
required rather than optional in the UI. `ip` and `message` are empty until
the job progresses far enough to know them.

> There is no `GET /api/v1/nodes/:id` (single node) yet — always fetch
> nodes through this cluster-scoped list.

---

## Nodes (worker management)

### `POST /api/v1/clusters/:id/nodes`

Adds a worker node to a cluster. Launches a VM on the cluster's network,
fetches a **fresh** join token from the master
(`kubeadm token create --print-join-command` — not the one `kubeadm init`
printed originally, which may be long expired), and runs `kubeadm join`.
Same async-job pattern as cluster creation.

**Preconditions** (checked before anything is created — `409` if not met):
- The cluster's `status` must be `"ready"`.
- The cluster's master node `status` must be `"running"`.

**Request:** entirely optional — `{}` or no body at all is valid and uses
every default.
```json
{ "cpu": 2, "memory": "2GiB", "disk": "20GiB" }
```
Same fields, same defaults/minimums/validation as cluster creation's
`cpu`/`memory`/`disk` (see the table above) — these are also enforced for
`kubeadm join`, not just `init`.

**Response `202`:**
```json
{
  "node": {
    "id": "d6da0b00-9b36-47e5-89e2-7a3904199423",
    "clusterId": "c0150a11-3a9b-4a96-8182-aacae98c33fd",
    "jobId": "a0ed32d1-408a-4376-bd8e-5d54e9126bff",
    "name": "worker-1",
    "incusName": "worker-d6da0b009b36",
    "role": "worker",
    "status": "creating",
    "message": "Node creation started",
    "createdAt": "2026-08-02T03:09:28.180597522+05:30",
    "updatedAt": "2026-08-02T03:09:28.188440653+05:30"
  }
}
```

`name` auto-increments per cluster: `worker-1`, `worker-2`, ... (based on a
count of existing workers — if you build worker deletion later, note this
numbering isn't collision-proof against gaps from deleted workers, though
no delete endpoint exists yet so it's a non-issue today).

Poll `GET /api/v1/jobs/:jobId`; on success the job's `message` becomes
`"Node joined the cluster"` and the node's `status` becomes `"running"`.

**Errors:**
- `404` `"cluster not found"`
- `409` `"cluster not ready"` — wait for the cluster's master to finish first
- `409` `"master not running"` — same idea, different point of failure
- `400` — `cpu`/`memory`/`disk` below minimum
- `500` — database/job-creation error

> There is no way to remove a worker yet (no `DELETE`). Deleting a
> cluster/node (stopping and removing the underlying Incus VM, not just the
> DB row) is unimplemented — don't build that UI yet either.

---

## Jobs

Read-only — jobs are only ever created as a side effect of `POST
/api/v1/clusters` or `POST /api/v1/clusters/:id/nodes`. This is the primary
polling target for provisioning progress (see "The async job pattern").

### `GET /api/v1/jobs`

**Response `200`:** `{ "jobs": [ Job, ... ] }` (newest first, **all jobs
system-wide** — no filter by node/cluster; find the `jobId` you care about
via the node/cluster endpoints first, then poll it individually).

### `GET /api/v1/jobs/:id`

**Response `200`:**
```json
{
  "job": {
    "id": "a0ed32d1-408a-4376-bd8e-5d54e9126bff",
    "type": "node_provision",
    "name": "Provision worker node worker-d6da0b009b36",
    "status": "running",
    "progress": 80,
    "stage": "joining",
    "message": "Running kubeadm join...",
    "metadata": { "nodeId": "d6da0b00-9b36-47e5-89e2-7a3904199423", "role": "worker" },
    "createdAt": "2026-08-02T03:09:28.18648+05:30",
    "updatedAt": "2026-08-02T03:10:00.123456+05:30"
  }
}
```

On success, `status` becomes `"succeeded"`, `progress` is `100`, and
`result` is populated (currently just `{ "ip": "10.44.0.9" }`).

On failure:
```json
{
  "status": "failed",
  "stage": "failed",
  "message": "Node provisioning failed",
  "error": "command \"kubeadm init\" in instance \"master-...\" exited 1: ...(full kubeadm stderr)...",
  "completedAt": "2026-08-02T03:09:14.545476+05:30"
}
```
`error` is the raw underlying failure — often multi-line command output.
Fine to put in a collapsible "details" section, not meant as the headline
error message (use `message` for that).

`type` is currently always `"node_provision"` (used for both master and
worker provisioning — check `metadata.role` to distinguish which).

**Errors:** `404` `"job not found"`.

---

## Size string format

`memory` and `disk` fields use Incus's byte-size syntax: an integer
immediately followed by a unit suffix, no space.

- Decimal (powers of 1000): `B`, `kB`, `MB`, `GB`, `TB`, `PB`, `EB`
- Binary (powers of 1024): `KiB`, `MiB`, `GiB`, `TiB`, `PiB`, `EiB`

Examples: `"2GiB"`, `"1700MB"`, `"20GiB"`. An unparseable string (e.g.
`"lots"`, `"2 GB"` with a space, `"2gb"` wrong case) is rejected with `400`
and a message like `"memory: Invalid value: lots"`.

---

## Example: full create-cluster-with-worker flow

```
1. POST /api/v1/users                        {username}              → user.id
2. POST /api/v1/networks                     {ownerId, name, cidr}   → network.id
3. POST /api/v1/clusters                     {ownerId, networkId, name} → cluster.id (202, status: creating)
4. GET  /api/v1/clusters/:id/nodes                                    → nodes[0].jobId (the master)
5. poll GET /api/v1/jobs/:jobId  until status is succeeded|failed
6. GET  /api/v1/clusters/:id                                          → confirm status: ready
7. POST /api/v1/clusters/:id/nodes           {} (or sizing overrides) → node.id, node.jobId (202)
8. poll GET /api/v1/jobs/:jobId  until status is succeeded|failed
9. GET  /api/v1/clusters/:id/nodes                                    → confirm the new worker's status: running
```

---

## Known gaps (don't build UI expecting these yet)

- No authentication — `ownerId` is a bare, unvalidated-at-the-boundary UUID.
- No delete for Users, Clusters, or Nodes (only Cluster Networks support delete).
- No CNI is installed on any cluster, so `kubectl get nodes` (if you ever
  shell in) always shows every node as `NotReady`. There's no API surface
  for this either way — it's an operational fact, not something the API
  reports.
- No pagination on any list endpoint — expect small counts for now.
- No filtering by owner on list endpoints (`GET /networks`, `/clusters`
  return every owner's resources) — filter client-side if you need
  per-user views.
- Interrupted jobs (server restart mid-provisioning) are not recovered or
  retried — a job stuck in `"running"` after a backend restart is orphaned.
