CREATE TABLE IF NOT EXISTS nodes (
    id         TEXT PRIMARY KEY,
    cluster_id TEXT NOT NULL REFERENCES clusters (id) ON DELETE CASCADE,
    job_id     TEXT REFERENCES jobs (id) ON DELETE SET NULL,
    name       TEXT NOT NULL,
    incus_name TEXT NOT NULL,
    role       VARCHAR(10) NOT NULL,
    status     VARCHAR(20) NOT NULL DEFAULT 'creating',
    ip         TEXT,
    message    TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_nodes_role CHECK (role IN ('master', 'worker'))
);

-- Display name only needs to be unique within its own cluster (e.g. "master", "worker-1").
CREATE UNIQUE INDEX IF NOT EXISTS idx_nodes_cluster_name ON nodes (cluster_id, name);
-- incus_name is the real Incus VM instance name, which lives in a single
-- global namespace regardless of cluster/owner.
CREATE UNIQUE INDEX IF NOT EXISTS idx_nodes_incus_name ON nodes (incus_name);
CREATE INDEX IF NOT EXISTS idx_nodes_cluster_id ON nodes (cluster_id);

-- Enforce exactly one master node per cluster at the database level.
CREATE UNIQUE INDEX IF NOT EXISTS idx_nodes_one_master_per_cluster ON nodes (cluster_id) WHERE role = 'master';
