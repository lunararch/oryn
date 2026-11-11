-- +goose Up
CREATE TABLE desktop_clients (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    platform TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    registered_at INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL,
    revoked_at INTEGER
);

CREATE TABLE pairing_codes (
    id TEXT PRIMARY KEY,
    code_hash TEXT NOT NULL,
    expires_at INTEGER NOT NULL,
    used_at INTEGER,
    created_at INTEGER NOT NULL
);

ALTER TABLE play_sessions ADD COLUMN last_heartbeat_at INTEGER;

CREATE INDEX idx_play_sessions_client ON play_sessions (client_id);
CREATE INDEX idx_play_sessions_running_heartbeat ON play_sessions (status, last_heartbeat_at);

-- +goose Down
DROP INDEX IF EXISTS idx_play_sessions_running_heartbeat;
DROP INDEX IF EXISTS idx_play_sessions_client;
ALTER TABLE play_sessions DROP COLUMN last_heartbeat_at;
DROP TABLE IF EXISTS pairing_codes;
DROP TABLE IF EXISTS desktop_clients;
