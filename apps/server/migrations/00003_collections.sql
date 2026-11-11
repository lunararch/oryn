-- +goose Up
CREATE TABLE collections (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE collection_games (
    collection_id TEXT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    game_id TEXT NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    added_at INTEGER NOT NULL,
    PRIMARY KEY (collection_id, game_id)
);

CREATE INDEX idx_collection_games_game ON collection_games (game_id);

-- +goose Down
DROP TABLE IF EXISTS collection_games;
DROP TABLE IF EXISTS collections;
