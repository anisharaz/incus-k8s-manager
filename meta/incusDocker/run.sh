#!/usr/bin/env bash
#
# run.sh — Run the Incus (kii) container from the Docker Hub registry.
#
#   • Verifies Docker is installed and the daemon is accessible
#   • Checks whether the host supports KVM (and /dev/kvm is usable)
#   • Resolves the host's kvm group GID (KVM_GID)
#   • Runs the container in the HOST network namespace
#   • Pulls the image from Docker Hub and runs the container
#     (swappable for docker compose — see USE_COMPOSE)
#   • Waits for the container to become healthy
#
# Usage:
#   ./run.sh [--name NAME] [--image IMAGE] [--help]
#
set -euo pipefail

# ============================================================================
# Configuration
# ----------------------------------------------------------------------------
# To migrate to docker compose later:
#   1. Set USE_COMPOSE=true
#   2. Point COMPOSE_FILE at your docker-compose.yml
#   3. The compose file must mirror the same env / volumes / network settings.
#      Only run_container() below needs to change.
# ============================================================================
IMAGE_NAME="anisharaz/kii:latest"
CONTAINER_NAME="incus"
API_PORT="8443"          # port incusd listens on (exposed via host netns)
SETIPTABLES="true"
USE_COMPOSE=false
COMPOSE_FILE="docker-compose.yml"
HEALTH_TIMEOUT=300       # seconds to wait for "healthy"
HEALTH_INTERVAL=5        # health poll interval (seconds)

# Pretty output (disabled when not a tty)
if [ -t 1 ]; then
    GREEN=$'\e[0;32m'; YELLOW=$'\e[0;33m'; RED=$'\e[0;31m'; NC=$'\e[0m'
else
    GREEN=''; YELLOW=''; RED=''; NC=''
fi
info() { echo "${GREEN}[+]${NC} $*"; }
warn() { echo "${YELLOW}[!]${NC} $*"; }
fail() { echo "${RED}[x]${NC} $*" >&2; exit 1; }

usage() {
    cat <<EOF
Usage: $0 [options]

Options:
  --name NAME         Container name (default: $CONTAINER_NAME).
  --image IMAGE       Image tag (default: $IMAGE_NAME).
  -h, --help          Show this help.

The container always runs in the HOST network namespace.
EOF
}

# --- Argument parsing --------------------------------------------------------
while [ $# -gt 0 ]; do
    case "$1" in
        --name)            CONTAINER_NAME="${2:?--name needs a value}"; shift 2 ;;
        --image)           IMAGE_NAME="${2:?--image needs a value}"; shift 2 ;;
        -h|--help)         usage; exit 0 ;;
        *)                 fail "Unknown option: $1 (see --help)" ;;
    esac
done

# --- 1. Docker ---------------------------------------------------------------
check_docker() {
    info "Checking Docker..."
    command -v docker >/dev/null 2>&1 \
    || fail "Docker is not installed or not in PATH."
    docker info >/dev/null 2>&1 \
    || fail "Docker daemon is not accessible. Is it running? (systemctl start docker, or run as root / add user to docker group)."
    info "Docker is installed and the daemon is accessible."
}

# --- 2. KVM ------------------------------------------------------------------
KVM_AVAILABLE=false
KVM_GID=""

check_kvm() {
    info "Checking KVM support..."
    if [ -e /dev/kvm ]; then
        if [ -r /dev/kvm ] && [ -w /dev/kvm ]; then
            info "/dev/kvm is present and accessible."
            KVM_AVAILABLE=true
        else
            warn "/dev/kvm exists but is NOT accessible by this user (permissions)."
        fi
    else
        warn "/dev/kvm does not exist — KVM/VMs will not work."
        warn "Enable virtualization in BIOS, or nested virt on a VM host."
    fi
    
    if grep -qE 'vmx|svm' /proc/cpuinfo; then
        info "CPU virtualization extensions detected (vmx/svm)."
    else
        warn "No vmx/svm flags in /proc/cpuinfo — KVM may be unavailable."
    fi
}

get_kvm_gid() {
    [ "$KVM_AVAILABLE" = true ] || return 0
    KVM_GID="$(getent group kvm | cut -d: -f3 || true)"
    if [ -z "$KVM_GID" ]; then
        KVM_GID="$(stat -c '%g' /dev/kvm 2>/dev/null || true)"
    fi
    if [ -n "$KVM_GID" ]; then
        info "Using KVM_GID=$KVM_GID"
    else
        warn "Could not determine the kvm group GID — omitting KVM_GID."
    fi
}

# --- 3. Pull & run -----------------------------------------------------------
pull_image() {
    [ "$USE_COMPOSE" = true ] && return 0
    info "Pulling image '$IMAGE_NAME' from Docker Hub..."
    docker pull "$IMAGE_NAME" || fail "docker pull failed for '$IMAGE_NAME'."
}

run_container() {
    # Remove any previous container with the same name for a clean start.
    if docker ps -a --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
        warn "Removing existing container '$CONTAINER_NAME'."
        docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    if [ "$USE_COMPOSE" = true ]; then
        # ---- docker compose path --------------------------------------------
        # docker-compose.yml mirrors the same options below
        # (network_mode, KVM_GID, SETIPTABLES, /dev, /lib/modules).
        info "Starting via docker compose ($COMPOSE_FILE)..."
        # Pass the resolved values through so compose ${VAR} interpolation
        # picks them up (KVM_GID may be empty -> compose uses its default).
        export KVM_GID SETIPTABLES
        docker compose -f "$COMPOSE_FILE" up -d
        return 0
    fi
    
    # ---- docker run path ---------------------------------------------------
    local -a cmd=(docker run -d --name "$CONTAINER_NAME" --privileged --restart unless-stopped)
    [ -n "$KVM_GID" ] && cmd+=(-e "KVM_GID=$KVM_GID")
    cmd+=(-e "SETIPTABLES=$SETIPTABLES")
    cmd+=(-v /dev:/dev -v /lib/modules:/lib/modules:ro)
    
    info "Using host network namespace (API on port $API_PORT)."
    cmd+=(--network host)
    cmd+=("$IMAGE_NAME")
    
    info "Running: ${cmd[*]}"
    "${cmd[@]}" || fail "docker run failed for container '$CONTAINER_NAME'."
}

# --- 5. Wait for healthy -----------------------------------------------------
wait_healthy() {
    info "Waiting for '$CONTAINER_NAME' to become healthy (timeout ${HEALTH_TIMEOUT}s)..."
    local waited=0 status=""
    while [ "$waited" -lt "$HEALTH_TIMEOUT" ]; do
        if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
            if docker ps -a --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
                fail "Container '$CONTAINER_NAME' stopped. Logs:\n$(docker logs --tail 50 "$CONTAINER_NAME" 2>&1)"
            fi
            fail "Container '$CONTAINER_NAME' is not present."
        fi
        
        status="$(docker inspect -f '{{.State.Health.Status}}' "$CONTAINER_NAME" 2>/dev/null || echo starting)"
        case "$status" in
            healthy)
                info "Container is healthy after ~${waited}s."
                return 0
            ;;
            unhealthy)
                fail "Container is unhealthy. Logs:\n$(docker logs --tail 50 "$CONTAINER_NAME" 2>&1)"
            ;;
        esac
        sleep "$HEALTH_INTERVAL"
        waited=$((waited + HEALTH_INTERVAL))
    done
    fail "Timed out waiting for '$CONTAINER_NAME' to become healthy. Logs:\n$(docker logs --tail 50 "$CONTAINER_NAME" 2>&1)"
}

# --- Summary -----------------------------------------------------------------
print_summary() {
    echo
    info "Incus container is up and healthy!"
    echo "  Container : $CONTAINER_NAME"
    echo "  Image     : $IMAGE_NAME"
    echo "  Network   : host"
    echo "  API URL   : https://localhost:$API_PORT  (self-signed cert)"
    echo "  CLI       : docker exec -it $CONTAINER_NAME incus list"
    echo "  Logs      : docker logs -f $CONTAINER_NAME"
}

# --- Main --------------------------------------------------------------------
main() {
    check_docker
    check_kvm
    get_kvm_gid
    pull_image
    run_container
    wait_healthy
    print_summary
}
main "$@"
