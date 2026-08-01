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
#   3. Stages the VM image files into a temp dir (/tmp), exposed to docker
#      as the "vmimage" named build context, and deleted after the build.
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
# Temp staging dir for the VM image files; passed to docker as the
# "vmimage" named build context and removed automatically on exit.
STAGE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/kii-vm.XXXXXX")"

cleanup() { rm -rf "$STAGE_DIR"; }
trap cleanup EXIT

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

# --- 3. Stage the VM image files into the temp dir ----------------------------
for f in incus.tar.xz disk.qcow2; do
    info "Staging $f into $STAGE_DIR..."
    cp -f "$VM_DIR/$f" "$STAGE_DIR/$f"
done

# --- 4. Docker build ----------------------------------------------------------
info "Building image '$IMAGE_NAME'..."
# Pass the temp dir as the "vmimage" named build context so the Dockerfile
# can COPY the VM image files without them living in the repo/build context.
docker build "${BUILD_ARGS[@]}" \
--build-context "vmimage=$STAGE_DIR" \
-t "$IMAGE_NAME" "$BUILD_CONTEXT"
info "Done. Image: $IMAGE_NAME"
