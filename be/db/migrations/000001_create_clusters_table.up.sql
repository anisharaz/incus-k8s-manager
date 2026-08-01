CREATE TABLE IF NOT EXISTS clusters (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    status     VARCHAR(50) NOT NULL DEFAULT 'creating',
    job_id     TEXT,
    ip         TEXT,
    message    TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_clusters_name ON clusters (name);
