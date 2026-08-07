#!/usr/bin/env bash
# Downloads docker-compose.yml, generates a .env with an auto-detected
# KVM_GID, and starts the incus-k8s-manager stack. Safe to re-run — it
# won't overwrite an existing .env.
set -euo pipefail

COMPOSE_URL="https://raw.githubusercontent.com/anisharaz/incus-k8s-manager/refs/heads/main/meta/incusDocker/docker-compose.yml"
DIR="${1:-incus-k8s-manager}"

mkdir -p "$DIR"
cd "$DIR"

echo "==> Downloading docker-compose.yml"
curl -fsSL "$COMPOSE_URL" -o docker-compose.yml

if [ ! -f .env ]; then
  echo "==> Detecting the host's kvm group"
  KVM_GID="$(getent group kvm 2>/dev/null | cut -d: -f3 || true)"
  if [ -z "${KVM_GID:-}" ] && [ -e /dev/kvm ]; then
    KVM_GID="$(stat -c '%g' /dev/kvm)"
  fi
  if [ -z "${KVM_GID:-}" ]; then
    echo "    Could not detect a kvm group — defaulting to 0. If VM creation" >&2
    echo "    fails, find your host's kvm GID and set KVM_GID in .env yourself." >&2
    KVM_GID=0
  fi

  cat > .env <<EOF
KVM_GID=$KVM_GID
SETIPTABLES=true

POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=incus_k8s_manager
# JWT_SECRET=

# The bundled proxy service terminates TLS in front of the app.
COOKIE_SECURE=true
EOF
  echo "==> Wrote .env (KVM_GID=$KVM_GID)"
else
  echo "==> .env already exists, leaving it as-is"
fi

echo "==> Pulling images"
docker compose pull

echo "==> Starting the stack"
docker compose up -d

cat <<'EOF'

Done. Once the containers report healthy (docker compose ps), open:

  https://localhost   (or https://<this-server's-IP> from another machine)

The stack terminates TLS itself with a self-signed certificate (no domain
needed), so your browser will show a "not secure" warning the first time —
that's expected, click through it. The connection is still genuinely
encrypted; only the certificate is unverified by a public CA.

EOF
