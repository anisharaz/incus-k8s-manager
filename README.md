# incus-k8s-manager

Spin up Kubernetes clusters on [Incus](https://linuxcontainers.org/incus/) VMs from a web UI, no manual `kubeadm` babysitting required.

## What it is

A self-hosted manager for creating and growing Kubernetes clusters backed by Incus VMs. Define a network, create a cluster on it, and the master node is provisioned and `kubeadm init`-ed automatically as a background job. Add worker nodes afterward and they join the cluster on their own. Everything — networks, clusters, nodes — is scoped to the logged-in user, with a simple admin/user role model.

## How it works

- **Backend** (`be/`) — a Go/Fiber API backed by PostgreSQL. It talks to the Incus daemon over its Unix socket to launch and configure VMs, and tracks long-running provisioning as DB-persisted background jobs the frontend polls for progress.
- **Frontend** (`fe/`) — a React/Vite/shadcn UI for logging in, managing networks, and creating/inspecting clusters and nodes.
- **Incus image** (`meta/incusDocker/`) — a Docker image that runs `incusd` itself, preloaded with a Kubernetes-ready VM image, so the whole stack can run anywhere Docker + KVM does.

The long-term goal is a single Docker container bundling all three, so running a Kubernetes lab is a `docker run` away.

## Project layout

```
be/    Go backend (API, job manager, Incus client)
fe/    React frontend
meta/  Incus-in-Docker image used as the VM host
API.md backend REST API reference
```

## Getting started

See [`be/README.md`](be/README.md) for backend setup (Postgres, migrations, env vars) and [`meta/incusDocker/README.md`](meta/incusDocker/README.md) for standing up the Incus daemon. The frontend is a standard `pnpm install && pnpm dev` against `fe/`, proxied to the backend in dev.

## Docs

- [`API.md`](API.md) — full REST API reference (auth, networks, clusters, nodes, jobs).
- [`be/README.md`](be/README.md) — backend architecture and internals.
- [`meta/incusDocker/README.md`](meta/incusDocker/README.md) — building/running the Incus daemon image.

## Roadmap / TODO

- [ ] **CNI installation** — auto-install a pod network add-on (e.g. Calico/Flannel) as the last step of master provisioning, so clusters are actually usable (`Ready`) without a manual step.
- [ ] **Cluster/node deletion** — no way to tear down a cluster or remove a worker node yet.
- [ ] **Kubeconfig download** — expose an endpoint/UI action to download a cluster's `admin.conf` instead of requiring a manual VM exec.
- [ ] **Live job progress via WebSocket** — replace polling with push updates for job status.
- [ ] **Job recovery on restart** — detect and resolve jobs left "running" in the DB if the backend restarts mid-provisioning.
- [ ] **Resource quotas** — cap per-user cluster/node counts or total CPU/memory to prevent exhausting the host.
