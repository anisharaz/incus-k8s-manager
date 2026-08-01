CREATE TABLE IF NOT EXISTS jobs (
    id           TEXT PRIMARY KEY,
    type         TEXT NOT NULL,
    name         TEXT,
    status       VARCHAR(20) NOT NULL DEFAULT 'queued',
    progress     INTEGER NOT NULL DEFAULT 0,
    stage        TEXT,
    message      TEXT,
    result       JSONB,
    error        TEXT,
    metadata     JSONB,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_jobs_type ON jobs (type);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs (status);
