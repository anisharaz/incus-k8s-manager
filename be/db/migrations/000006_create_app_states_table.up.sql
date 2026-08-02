-- Singleton row (id is always 1) tracking app-wide bootstrap state, so the
-- API can tell the frontend whether to show "register admin" or "log in".
CREATE TABLE IF NOT EXISTS app_states (
    id            INTEGER PRIMARY KEY DEFAULT 1,
    admin_created BOOLEAN NOT NULL DEFAULT false,
    CONSTRAINT chk_app_states_singleton CHECK (id = 1)
);

INSERT INTO app_states (id, admin_created)
VALUES (1, false)
ON CONFLICT (id) DO NOTHING;
