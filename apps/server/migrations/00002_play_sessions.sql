-- +goose Up
CREATE TABLE play_sessions (
    id TEXT PRIMARY KEY,
    game_id TEXT NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    client_id TEXT,
    started_at INTEGER NOT NULL,
    ended_at INTEGER,
    duration_seconds INTEGER NOT NULL DEFAULT 0 CHECK (duration_seconds >= 0),
    status TEXT NOT NULL
        CHECK (status IN ('running', 'completed', 'interrupted', 'failed')),
    source TEXT NOT NULL CHECK (source IN ('desktop', 'manual')),
    exit_code INTEGER,
    notes TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX idx_play_sessions_game_started ON play_sessions (game_id, started_at DESC);
CREATE INDEX idx_play_sessions_status ON play_sessions (status);
CREATE INDEX idx_play_sessions_started ON play_sessions (started_at DESC);

-- +goose Down
DROP TABLE IF EXISTS play_sessions;
