-- +goose Up
CREATE TABLE libraries (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    root_path TEXT NOT NULL UNIQUE,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    last_scan_at INTEGER
);

CREATE TABLE games (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    relative_path TEXT NOT NULL,
    folder_name TEXT NOT NULL,
    detected_title TEXT NOT NULL,
    detected_description TEXT NOT NULL DEFAULT '',
    detected_developer TEXT NOT NULL DEFAULT '',
    detected_publisher TEXT NOT NULL DEFAULT '',
    detected_release_date TEXT NOT NULL DEFAULT '',
    detected_version TEXT NOT NULL DEFAULT '',
    detection_confidence INTEGER NOT NULL DEFAULT 0,
    detection_evidence TEXT NOT NULL DEFAULT '[]',
    availability_status TEXT NOT NULL DEFAULT 'available'
        CHECK (availability_status IN ('available', 'missing')),
    filesystem_fingerprint TEXT NOT NULL DEFAULT '',
    first_seen_at INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    UNIQUE (library_id, relative_path)
);

CREATE INDEX idx_games_library_availability ON games (library_id, availability_status);
CREATE INDEX idx_games_title ON games (detected_title);

CREATE TABLE game_overrides (
    game_id TEXT PRIMARY KEY REFERENCES games(id) ON DELETE CASCADE,
    title TEXT,
    description TEXT,
    developer TEXT,
    publisher TEXT,
    release_date TEXT,
    rating INTEGER CHECK (rating IS NULL OR rating BETWEEN 1 AND 10),
    notes TEXT,
    completion_status TEXT
        CHECK (completion_status IS NULL OR completion_status IN
            ('unplayed', 'playing', 'completed', 'dropped', 'on_hold')),
    updated_at INTEGER NOT NULL
);

CREATE TABLE favorites (
    game_id TEXT PRIMARY KEY REFERENCES games(id) ON DELETE CASCADE,
    created_at INTEGER NOT NULL
);

CREATE TABLE artwork (
    id TEXT PRIMARY KEY,
    game_id TEXT NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('cover', 'banner')),
    origin TEXT NOT NULL CHECK (origin IN ('detected', 'upload')),
    source_rel_path TEXT,
    cache_key TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    UNIQUE (game_id, kind)
);

CREATE TABLE scan_runs (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    status TEXT NOT NULL
        CHECK (status IN ('running', 'completed', 'partial', 'failed', 'cancelled')),
    started_at INTEGER NOT NULL,
    finished_at INTEGER,
    discovered_count INTEGER NOT NULL DEFAULT 0,
    added_count INTEGER NOT NULL DEFAULT 0,
    updated_count INTEGER NOT NULL DEFAULT 0,
    unchanged_count INTEGER NOT NULL DEFAULT 0,
    missing_count INTEGER NOT NULL DEFAULT 0,
    error_count INTEGER NOT NULL DEFAULT 0,
    summary TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_scan_runs_library_started ON scan_runs (library_id, started_at DESC);

CREATE TABLE scan_errors (
    id TEXT PRIMARY KEY,
    scan_run_id TEXT NOT NULL REFERENCES scan_runs(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    kind TEXT NOT NULL,
    message TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS scan_errors;
DROP TABLE IF EXISTS scan_runs;
DROP TABLE IF EXISTS artwork;
DROP TABLE IF EXISTS favorites;
DROP TABLE IF EXISTS game_overrides;
DROP TABLE IF EXISTS games;
DROP TABLE IF EXISTS libraries;
