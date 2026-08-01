#!/usr/bin/env bash
#
# build.sh — Build the KII (Incus) container image, baking in the
# prebuilt k8s VM image (incus.tar.xz + disk.qcow2).
#
#   1. Checks that distrobuilder is available (only needed if the VM
#      image files are missing).
#   2. Ensures incusStuff/incus.tar.xz and incusStuff/disk.qcow2 exist;
#      if not, runs:
#        sudo distrobuilder build-incus incus_distrobuilder.yaml --vm .
#   3. Stages the VM image files into this build context (images/).
#   4. Runs `docker build` to produce anisharaz/kii:latest.
#
# Usage:
#   ./build.sh
#   IMAGE_NAME=anisharaz/kii:v1.0.0 ./build.sh
#   ./build.sh --no-cache
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VM_DIR="$SCRIPT_DIR/incusStuff"   # distrobuilder inputs & VM image outputs

DISTRO_YAML="$VM_DIR/incus_distrobuilder.yaml"
VM_TAR="$VM_DIR/incus.tar.xz"
VM_DISK="$VM_DIR/disk.qcow2"

IMAGE_NAME="${IMAGE_NAME:-anisharaz/kii:latest}"
BUILD_CONTEXT="$SCRIPT_DIR"          # meta/incusDocker
STAGE_DIR="$BUILD_CONTEXT/images"    # staged copies used by the Dockerfile

BUILD_ARGS=()

info() { echo "[+] $*"; }
warn() { echo "[!] $*"; }
fail() { echo "[x] $*" >&2; exit 1; }

# --- Argument parsing --------------------------------------------------------
while [ $# -gt 0 ]; do
    case "$1" in
        --no-cache) BUILD_ARGS+=(--no-cache); shift ;;
        -h|--help)
            echo "Usage: $0 [--no-cache]"
            echo "  --no-cache   Pass --no-cache to docker build"
            exit 0
        ;;
        *) fail "Unknown option: $1 (see --help)" ;;
    esac
done

# --- 1. distrobuilder availability -------------------------------------------
if [ ! -f "$VM_TAR" ] || [ ! -f "$VM_DISK" ]; then
    if ! command -v distrobuilder >/dev/null 2>&1; then
        fail "distrobuilder is not installed. Install it (e.g. sudo pacman -S distrobuilder) or place incus.tar.xz and disk.qcow2 in $VM_DIR."
    fi
fi

# --- 2. Ensure the VM image files exist --------------------------------------
if [ -f "$VM_TAR" ] && [ -f "$VM_DISK" ]; then
    info "VM image files found:"
    info "  $VM_TAR"
    info "  $VM_DISK"
else
    warn "VM image files are missing. Building them with distrobuilder (this may take a while)..."
    ( cd "$VM_DIR" && sudo distrobuilder build-incus "$DISTRO_YAML" --vm . )
    [ -f "$VM_TAR" ] && [ -f "$VM_DISK" ] \
    || fail "distrobuilder did not produce both incus.tar.xz and disk.qcow2."
fi

# --- 3. Stage the VM image files into the build context -----------------------
mkdir -p "$STAGE_DIR"
for f in incus.tar.xz disk.qcow2; do
    src="$VM_DIR/$f"
    dst="$STAGE_DIR/$f"
    if [ ! -f "$dst" ] || [ "$src" -nt "$dst" ]; then
        info "Staging $f into build context..."
        cp -f "$src" "$dst"
    else
        info "$f already staged (up to date)."
    fi
done

# --- 4. Docker build ----------------------------------------------------------
info "Building image '$IMAGE_NAME'..."
docker build "${BUILD_ARGS[@]}" -t "$IMAGE_NAME" "$BUILD_CONTEXT"
info "Done. Image: $IMAGE_NAME"
