CREATE TABLE IF NOT EXISTS cluster_networks (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    cidr       TEXT NOT NULL,
    gateway    TEXT NOT NULL,
    status     VARCHAR(20) NOT NULL DEFAULT 'creating',
    message    TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_cluster_networks_name ON cluster_networks (name);
