CREATE TABLE IF NOT EXISTS cluster_networks (
    id         TEXT PRIMARY KEY,
    owner_id   TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    incus_name TEXT NOT NULL,
    cidr       TEXT NOT NULL,
    gateway    TEXT NOT NULL,
    status     VARCHAR(20) NOT NULL DEFAULT 'creating',
    message    TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Display name only needs to be unique within its owner's own resources.
CREATE UNIQUE INDEX IF NOT EXISTS idx_cluster_networks_owner_name ON cluster_networks (owner_id, name);
-- incus_name is the real Incus bridge interface name, which lives in a single
-- global namespace regardless of owner.
CREATE UNIQUE INDEX IF NOT EXISTS idx_cluster_networks_incus_name ON cluster_networks (incus_name);
CREATE INDEX IF NOT EXISTS idx_cluster_networks_owner_id ON cluster_networks (owner_id);
