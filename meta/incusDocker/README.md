# Incus Container Image (KOI)

This folder contains everything needed to build a containerized [Incus](https://linuxcontainers.org/incus/) server based on `debian:bookworm-slim`. The image runs `incusd` inside a Docker/Podman container and exposes the Incus REST API over HTTPS on port **8443**.

## Contents

| File                      | Description                                                                                                                                                          |
| ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Dockerfile`              | Builds the image, installs Incus + dependencies, copies `entrypoint.sh`, the preseed config, and bakes in the prebuilt k8s VM image (`incus.tar.xz` + `disk.qcow2`). |
| `entrypoint.sh`           | Entrypoint: starts `lxcfs`, `udevd`, `incusd`, then runs the preseed **and imports the k8s VM image** on first start (tracked by a marker in `/var/lib/incus`).      |
| `incus_admin_config.yaml` | Admin config: HTTPS on `:8443`, `default` dir pool, `clustermanagerbr0` bridge (NAT), default profile + project.                                                     |
| `run_app.sh`              | Host-side helper: validates Docker/KVM, resolves `KVM_GID`, writes `.env`, and runs `docker compose up -d --wait` for the whole stack (`incus`, `postgres`, `app`). |
| `docker-compose.yml`      | Compose definition for the full stack: `incus` (privileged, `KVM_GID`, `SETIPTABLES`, `/dev` + `/lib/modules` mounts, `incus-data` volume), `postgres`, `app`, and `proxy` (Caddy, terminates TLS). |
| `build_docker_image.sh`   | Recommended build wrapper: ensures `incusStuff/incus.tar.xz`/`disk.qcow2` exist (runs distrobuilder if needed), stages them, then runs `docker build`.               |
| `incusStuff/`             | Source folder for the k8s VM image: `incus_distrobuilder.yaml` (distrobuilder config) plus the built `incus.tar.xz` + `disk.qcow2`.                                  |

---

## Building the Image

The image is built with `build_docker_image.sh`, which:

1. Checks that `distrobuilder` is installed (only needed if the VM image is missing).
2. Ensures `incusStuff/incus.tar.xz` and `incusStuff/disk.qcow2` exist — if not, runs:
   ```bash
   sudo distrobuilder build-incus incusStuff/incus_distrobuilder.yaml --vm .
   ```
   (this produces the prebuilt Ubuntu + Kubernetes VM image).
3. Stages the two files into `images/` (the docker build context).
4. Runs `docker build -t aaraz/koivmrunner:latest .`

```bash
# From the meta/incusDocker directory
cd meta/incusDocker

./build_docker_image.sh                       # ensure VM image, then build
./build_docker_image.sh --no-cache            # bypass docker build cache
IMAGE_NAME=aaraz/koivmrunner:v1.0.0 ./build_docker_image.sh   # custom tag
```

> **Note:** the build context is the `meta/incusDocker` folder, so `entrypoint.sh`, `incus-preseed.yaml`, and the staged VM image files (`incus.tar.xz` + `disk.qcow2`, staged into a temp dir by `build_docker_image.sh`) are picked up automatically by the `COPY` instructions in the `Dockerfile`.

If you already have `incus.tar.xz` and `disk.qcow2` in `incusStuff/`, you can also build directly:

```bash
# Requires the VM image files already staged (build_docker_image.sh does this automatically)
docker build -t aaraz/koivmrunner:latest .

# Build with a different tag
docker build -t aaraz/koivmrunner:v1.0.0 .
```

---

## Quick Start: `run_app.sh` Helper Script

`run_app.sh` automates the whole stack (`incus` + `postgres` + `app`) for you — it:

1. Checks that Docker (or `docker-compose`) is installed and the daemon is accessible.
2. Checks that `/dev/kvm` is available, and warns if virtualization extensions aren't detected.
3. Resolves the host's `kvm` group GID automatically and writes it into `.env`.
4. Runs `docker compose up -d --wait` (with a 5-minute timeout for the stack to report healthy).

```bash
./run_app.sh
```

No flags — it always drives the `docker-compose.yml` in this directory as-is. For anything more targeted (pulling/running just the Incus image standalone, a custom container name), use `docker compose`/`docker run` directly — see the next two sections.

---

## Docker Compose

`docker-compose.yml` brings up the full stack: the `incus` daemon (mirrors the `docker run` options above, always on the **host** network namespace), a `postgres` database, `app` — the backend itself, built from `../../Dockerfile` — and `proxy`, a Caddy container that terminates TLS in front of `app` (see [HTTPS](#https) below). `app` waits on both `postgres`'s and `incus`'s healthchecks before starting. `incus`'s healthcheck overrides the image's default (which only checks that `incusd` answers) to also wait for `entrypoint.sh`'s one-time preseed and k8s VM image import to finish — `app` reaching the daemon before then would see no network/profile and no `k8s` image, and cluster creation would fail. `app` applies any pending database migrations itself on every startup (see `be/cmd/server/main.go`'s `runMigrations`) — there's no separate migration step to run. `app` has no published port of its own; `proxy` is the only way in from outside the compose network. Environment variables are configured in a `.env` file that `docker compose` auto-loads from this directory.

First, check and adjust `KVM_GID` in `.env` for **your** host (see below), then:

```bash
cd meta/incusDocker

# Start (--build picks up any be/ source changes)
docker compose up -d --build

# Wait until incus reports "healthy" and postgres/app are "running"
docker compose ps

# Logs
docker compose logs -f incus
docker compose logs -f app

# Stop & remove
docker compose down
```

The API is then reachable at `https://localhost` (e.g. `curl -k https://localhost/api/v1/auth/status` — `-k` because the cert is self-signed by default, see below).

### Environment via `.env`

The `.env` file contains:

```bash
KVM_GID=990        # host's kvm group GID (for VM support)
SETIPTABLES=true   # add ACCEPT rules to DOCKER-USER iptables chains

POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=incus_k8s_manager
# JWT_SECRET=       # set this for any deployment that should survive a restart
                     # without invalidating sessions — see be/README.md
COOKIE_SECURE=true   # keep true as long as `proxy` (or some other TLS
                     # terminator) sits in front of `app` — see HTTPS below
```

`docker compose` reads `.env` automatically, so a plain `docker compose up -d` is all you need. To change a value, edit `.env` and run `docker compose up -d` again (Compose recreates the container with the new env).

> The compose file also defines a named volume `postgres-data` mounted at `/var/lib/postgresql/data`, so the database survives `docker compose down`/`up` (use `docker compose down -v` to also wipe it).

> The Incus API listens on port **8443** directly on the host (no `-p` port mapping is needed with host networking).

> The compose file also defines a named volume `incus-data` mounted at `/var/lib/incus`. It persists Incus state **and** the preseed marker across restarts/recreates, so `entrypoint.sh` only runs `incus admin init --preseed` on the very first start.

To find the correct `KVM_GID` for your system (it differs per distro — e.g. `990`, `78`, `36`):

```bash
getent group kvm          # kvm:x:990:anish  -> GID is 990
# or
stat -c '%g' /dev/kvm     # 990
```

Then set that number in `.env`:

```bash
sed -i "s/^KVM_GID=.*/KVM_GID=$(getent group kvm | cut -d: -f3)/" .env
```

> Shell exports take priority over `.env` in Compose, so a one-off override is still possible: `SETIPTABLES=false docker compose up -d`.

## HTTPS

A session cookie marked `Secure` (which this app always sets when
`COOKIE_SECURE=true`) is only stored/sent by browsers over a real HTTPS
connection — **except** for `localhost`/`127.0.0.1`, which browsers treat
as secure even over plain HTTP. That's why logging in works fine at
`http://localhost:8000` but silently fails (login "succeeds" but nothing
stays logged in) at `http://<server-ip>:8000` — the browser just discards
the cookie. So this stack terminates real HTTPS itself, in front of `app`,
rather than ever running over plain HTTP outside of `localhost`.

The `proxy` service (`caddy:2-alpine`) does this with **no separate config
file to download** — the whole Caddyfile is inlined in `docker-compose.yml`
itself, under the top-level `configs:` key (Compose's file-less config
content), and mounted into the container at `/etc/caddy/Caddyfile`:

```
:443 {
  tls internal {
    on_demand
  }
  reverse_proxy app:8000
}
```

(plus a small same-host `:9080` block that unconditionally approves
on-demand certificate requests — required for `on_demand` to work at all,
see the comments above the `proxy` service for why.)

- `tls internal { on_demand }` mints a self-signed certificate from Caddy's
  own internal CA **the first time a given hostname/IP connects**, rather
  than for one fixed identifier — so it works equally for
  `https://localhost` and `https://<server-ip>` without knowing the
  server's address ahead of time. There's no domain name involved, so
  browsers will show a "not secure, proceed anyway" warning the first time
  — expected, and the connection is still genuinely encrypted (the cookie
  works correctly). `caddy-data` (a named volume) persists the CA and
  issued certs across restarts, so they aren't regenerated — and
  re-flagged as newly-untrusted — every time the container recreates.
- HTTP (`:80`) redirects to HTTPS automatically — Caddy does this by
  default whenever it's serving TLS.
- WebSocket connections (used by the in-browser node terminal) are proxied
  transparently by `reverse_proxy`, same as any other request.

**If you get a real domain name later**, edit the `caddyfile` config's
content in `docker-compose.yml`, replacing the `:443` block with:
```
yourdomain.com {
  reverse_proxy app:8000
}
```
(drop the `tls internal { on_demand }` block and the `:9080` ask endpoint
entirely) — Caddy then requests and renews a real Let's Encrypt certificate
automatically. Nothing in `app` needs to change either way; it only ever
speaks plain HTTP and has no idea whether `proxy` is using a self-signed or
a real certificate.

**If you'd rather front the stack with your own reverse proxy/load
balancer** (common once this moves behind a real ingress, e.g. a future
Kubernetes deployment), remove the `proxy` service, publish `app`'s port
8000 yourself, and set `COOKIE_SECURE` to match whatever your own
TLS-terminator does (`true` if it terminates TLS, `false` if you're
intentionally running over plain HTTP, e.g. an isolated LAN test).

---

## Running the Container

Basic privileged run (required for nested KVM, network bridging and device access). The container runs in the **host** network namespace:

```bash
docker run -d \
  --name incus \
  --privileged \
  --restart unless-stopped \
  --network host \
  -e SETIPTABLES=true \
  -e KVM_GID=<your_kvm_gid> \
  -v /dev:/dev \
  -v /lib/modules:/lib/modules:ro \
  aaraz/koivmrunner:latest
```

### Environment Variables

| Variable      | Default | Description                                                                                                                                  |
| ------------- | ------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| `SETIPTABLES` | `false` | When `true`, adds ACCEPT rules to the `DOCKER-USER` chains so container traffic isn't dropped.                                               |
| `KVM_GID`     | unset   | GID of the host `kvm` group. If set, the container's `kvm` group is remapped to match the host so VMs have the right `/dev/kvm` permissions. |

### Finding the correct KVM_GID

The `kvm` group's GID is **not** fixed — it can differ between systems and distros (e.g. `990`, `78`, `36`, ...). You must look it up on the **host** that will run the container:

```bash
# Option 1: getent
getent group kvm
# Example output: kvm:x:990:anish   -> GID is 990

# Option 2: grep /etc/group
grep '^kvm:' /etc/group
# Example output: kvm:x:990:

# Option 3: stat the device node (works if /dev/kvm exists)
stat -c '%G %g' /dev/kvm
# Example output: kvm 990
```

Then use that number in the run command:

```bash
KVM_GID=$(getent group kvm | cut -d: -f3)
echo "KVM_GID=$KVM_GID"

docker run -d \
  --name incus \
  --privileged \
  --restart unless-stopped \
  --network host \
  -e SETIPTABLES=true \
  -e KVM_GID="$KVM_GID" \
  -v /dev:/dev \
  -v /lib/modules:/lib/modules:ro \
  aaraz/koivmrunner:latest
```

> If you don't plan to run VMs (only containers), you can omit `KVM_GID` entirely.

---

## Using Incus

After the container starts, `entrypoint.sh` waits for `incusd` to be ready and then runs the preseed **only once** (tracked by a marker file at `/var/lib/incus/.preseed-done`). The preseed (`incus admin init --preseed < /incus-preseed.yaml`) configures:

- Listens on `:8443` over HTTPS (`core.https_address`)
- Creates the `default` storage pool (`dir` driver)
- Creates the `clustermanagerbr0` bridge with IPv4/IPv6 NAT
- Creates the `default` profile and `default` project

Right after the preseed, `entrypoint.sh` imports the prebuilt k8s VM image baked into the image at `/incus-images/`:

```bash
incus image import /incus-images/incus.tar.xz /incus-images/disk.qcow2 --alias k8s
```

So after the first start, an image with alias **`k8s`** is available to launch VMs:

```bash
incus launch k8s my-cluster-node
```

On later restarts the marker is found and the preseed + import steps are skipped. Persist `/var/lib/incus` with a volume (as `docker-compose.yml` does) so the marker — and all Incus data — survives container recreates.

### Quick checks

```bash
# Container status & logs
docker ps | grep incus
docker logs -f incus

# First start: you'll see the preseed line:
#   "Running incus admin init --preseed to complete setup..."
#   followed by "Importing k8s VM image (alias 'k8s')..."
# Later restarts: "Incus already initialized — skipping preseed."
```

### Using the incus CLI inside the container

```bash
docker exec -it incus incus list
docker exec -it incus incus network list
docker exec -it incus incus profile show default
docker exec -it incus incus image list   # should show the 'k8s' VM image
```

### Accessing the REST API from the host

The API listens on HTTPS port **8443** directly on the host (host network namespace). Since the cert is self-signed, use `--accept-certificate` the first time:

```bash
incus remote add my-incus https://<host-ip>:8443 --accept-certificate
incus remote switch my-incus
incus list
```

> The client must be run as `root` (or use `sudo`) to connect as an admin, or configure a trusted client certificate / authentication method first.

---

## Stopping & Cleaning Up

```bash
docker stop incus
docker rm incus

# Full cleanup (including build cache)
docker system prune -a
```

---

## Troubleshooting

| Problem                                           | Likely fix                                                                                                                                                                               |
| ------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Preseed fails with `network "incusbr0" not found` | The profile device name must match a defined network. This image uses `clustermanagerbr0`.                                                                                               |
| VMs fail with `/dev/kvm` permission errors        | Set `KVM_GID` to the host's kvm GID (see above).                                                                                                                                         |
| Containers can't reach the internet               | Ensure `SETIPTABLES=true` and that `ip_forward` is enabled on the host (`sysctl net.ipv4.ip_forward=1`).                                                                                 |
| Port 8443 already in use                          | With host networking the container binds `:8443` directly. Change the incus `core.https_address` in `incus-preseed.yaml` (e.g. `:9443`) and rebuild the image, or free up the host port. |
