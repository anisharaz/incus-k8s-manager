# Incus Container Image (kii)

This folder contains everything needed to build a containerized [Incus](https://linuxcontainers.org/incus/) server based on `debian:bookworm-slim`. The image runs `incusd` inside a Docker/Podman container and exposes the Incus REST API over HTTPS on port **8443**.

## Contents

| File                 | Description                                                                                                                                                     |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Dockerfile`         | Builds the image, installs Incus + dependencies, copies `start.sh`, the preseed config, and bakes in the prebuilt k8s VM image (`incus.tar.xz` + `disk.qcow2`). |
| `start.sh`           | Entrypoint: starts `lxcfs`, `udevd`, `incusd`, then runs the preseed **and imports the k8s VM image** on first start (tracked by a marker in `/var/lib/incus`). |
| `incus-preseed.yaml` | Preseed config: HTTPS on `:8443`, `default` dir pool, `clustermanagerbr0` bridge (NAT), default profile + project.                                              |
| `run.sh`             | Host-side helper: validates Docker/KVM, resolves `KVM_GID`, runs the container in the host netns and waits for it to become healthy.                            |
| `docker-compose.yml` | Compose definition mirroring the `run.sh` options (privileged, `KVM_GID`, `SETIPTABLES`, `/dev` + `/lib/modules` mounts, `incus-data` volume).                  |
| `build.sh`           | Recommended build wrapper: ensures `incusStuff/incus.tar.xz`/`disk.qcow2` exist (runs distrobuilder if needed), stages them, then runs `docker build`.          |
| `incusStuff/`        | Source folder for the k8s VM image: `incus_distrobuilder.yaml` (distrobuilder config) plus the built `incus.tar.xz` + `disk.qcow2`.                             |

---

## Building the Image

The image is built with `build.sh`, which:

1. Checks that `distrobuilder` is installed (only needed if the VM image is missing).
2. Ensures `incusStuff/incus.tar.xz` and `incusStuff/disk.qcow2` exist — if not, runs:
   ```bash
   sudo distrobuilder build-incus incusStuff/incus_distrobuilder.yaml --vm .
   ```
   (this produces the prebuilt Ubuntu + Kubernetes VM image).
3. Stages the two files into `images/` (the docker build context).
4. Runs `docker build -t anisharaz/kii:latest .`

```bash
# From the meta/incusDocker directory
cd meta/incusDocker

./build.sh                       # ensure VM image, then build
./build.sh --no-cache            # bypass docker build cache
IMAGE_NAME=anisharaz/kii:v1.0.0 ./build.sh   # custom tag
```

> **Note:** the build context is the `meta/incusDocker` folder, so `start.sh`, `incus-preseed.yaml`, and the staged `images/` (`incus.tar.xz` + `disk.qcow2`) are picked up automatically by the `COPY` instructions in the `Dockerfile`.

If you already have `incus.tar.xz` and `disk.qcow2` in `incusStuff/`, you can also build directly:

```bash
# Requires images/ already staged (build.sh does this automatically)
docker build -t anisharaz/kii:latest .

# Build with a different tag
docker build -t anisharaz/kii:v1.0.0 .
```

---

## Quick Start: `run.sh` Helper Script

`run.sh` automates the whole flow for you — it:

1. Checks that Docker is installed and the daemon is accessible.
2. Checks whether the host supports KVM (and that `/dev/kvm` is usable).
3. Resolves the host's `kvm` group GID automatically (`KVM_GID`).
4. Runs the container in the **host** network namespace.
5. Pulls the image from Docker Hub (`anisharaz/kii:latest`) if it isn't already present.
6. Waits until the container reports `healthy`.

> No Dockerfile or source files are needed on the user's machine — `run.sh` is self-contained and just fetches the prebuilt image from the registry.

```bash
./run.sh                 # run in the host network namespace
./run.sh --name incus --image anisharaz/kii:latest
```

CLI options:

| Option       | Description                                              |
| ------------ | -------------------------------------------------------- |
| `--name`     | Container name (default: `incus`).                       |
| `--image`    | Image tag to pull/run (default: `anisharaz/kii:latest`). |
| `-h, --help` | Show usage.                                              |

> To run through **docker compose** instead, set `USE_COMPOSE=true` at the top of `run.sh` (or run `docker compose up -d` directly — see the next section). When `USE_COMPOSE=true`, `run.sh` exports the resolved `KVM_GID`/`SETIPTABLES` so the compose file picks them up.

---

## Docker Compose

A `docker-compose.yml` is included that mirrors the `docker run` options above. It always uses the **host** network namespace (`network_mode: host`). Environment variables are configured in a `.env` file that `docker compose` auto-loads from this directory.

First, check and adjust `KVM_GID` in `.env` for **your** host (see below), then:

```bash
cd meta/incusDocker

# Start
docker compose up -d

# Wait until the container reports "healthy"
docker compose ps

# Logs
docker compose logs -f incus

# Stop & remove
docker compose down
```

### Environment via `.env`

The `.env` file contains:

```bash
KVM_GID=990        # host's kvm group GID (for VM support)
SETIPTABLES=true   # add ACCEPT rules to DOCKER-USER iptables chains
```

`docker compose` reads `.env` automatically, so a plain `docker compose up -d` is all you need. To change a value, edit `.env` and run `docker compose up -d` again (Compose recreates the container with the new env).

> The Incus API listens on port **8443** directly on the host (no `-p` port mapping is needed with host networking).

> The compose file also defines a named volume `incus-data` mounted at `/var/lib/incus`. It persists Incus state **and** the preseed marker across restarts/recreates, so `start.sh` only runs `incus admin init --preseed` on the very first start.

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
  anisharaz/kii:latest
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
  anisharaz/kii:latest
```

> If you don't plan to run VMs (only containers), you can omit `KVM_GID` entirely.

---

## Using Incus

After the container starts, `start.sh` waits for `incusd` to be ready and then runs the preseed **only once** (tracked by a marker file at `/var/lib/incus/.preseed-done`). The preseed (`incus admin init --preseed < /incus-preseed.yaml`) configures:

- Listens on `:8443` over HTTPS (`core.https_address`)
- Creates the `default` storage pool (`dir` driver)
- Creates the `clustermanagerbr0` bridge with IPv4/IPv6 NAT
- Creates the `default` profile and `default` project

Right after the preseed, `start.sh` imports the prebuilt k8s VM image baked into the image at `/incus-images/`:

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
