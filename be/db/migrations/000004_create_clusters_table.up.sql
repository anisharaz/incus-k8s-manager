CREATE TABLE IF NOT EXISTS clusters (
    id         TEXT PRIMARY KEY,
    owner_id   TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    network_id TEXT NOT NULL REFERENCES cluster_networks (id) ON DELETE RESTRICT,
    name       TEXT NOT NULL,
    status     VARCHAR(20) NOT NULL DEFAULT 'creating',
    message    TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- A cluster isn't itself an Incus resource (its nodes are), so it only needs
-- a name unique within its owner's own resources.
CREATE UNIQUE INDEX IF NOT EXISTS idx_clusters_owner_name ON clusters (owner_id, name);
CREATE INDEX IF NOT EXISTS idx_clusters_owner_id ON clusters (owner_id);
CREATE INDEX IF NOT EXISTS idx_clusters_network_id ON clusters (network_id);
