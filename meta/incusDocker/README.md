# Incus Container Image (KOI)

This folder contains everything needed to build a containerized [Incus](https://linuxcontainers.org/incus/) server based on `debian:bookworm-slim`. The image runs `incusd` inside a Docker/Podman container and exposes the Incus REST API over HTTPS on port **8443**.

## Contents

| File                      | Description                                                                                                                                                          |
| ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Dockerfile`              | Builds the image, installs Incus + dependencies, copies `entrypoint.sh`, the preseed config, and bakes in the prebuilt k8s VM image (`incus.tar.xz` + `disk.qcow2`). |
| `entrypoint.sh`           | Entrypoint: starts `lxcfs`, `udevd`, `incusd`, then runs the preseed **and imports the k8s VM image** on first start (tracked by a marker in `/var/lib/incus`).      |
| `incus_admin_config.yaml` | Admin config: HTTPS on `:8443`, `default` dir pool, `clustermanagerbr0` bridge (NAT), default profile + project.                                                     |
| `run_app.sh`              | Host-side helper: validates Docker/KVM, resolves `KVM_GID`, writes `.env`, and runs `docker compose up -d --wait` for the whole stack (`incus`, `postgres`, `app`). |
| `docker-compose.yml`      | Compose definition for the full stack: `incus` (privileged, `KVM_GID`, `SETIPTABLES`, `/dev` + `/lib/modules` mounts, `incus-data` volume), `postgres`, and `app`.   |
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

`docker-compose.yml` brings up the full stack: the `incus` daemon (mirrors the `docker run` options above, always on the **host** network namespace), a `postgres` database, and `app` — the backend itself, built from `../../be/Dockerfile`. `app` waits on `postgres`'s healthcheck before starting, and applies any pending database migrations itself on every startup (see `be/cmd/server/main.go`'s `runMigrations`) — there's no separate migration step to run. Environment variables are configured in a `.env` file that `docker compose` auto-loads from this directory.

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

The API is then reachable at `http://localhost:8000` (e.g. `curl localhost:8000/api/v1/auth/status`).

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
