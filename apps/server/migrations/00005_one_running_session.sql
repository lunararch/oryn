-- +goose Up
CREATE UNIQUE INDEX idx_play_sessions_one_running_per_client
    ON play_sessions (client_id)
    WHERE status = 'running' AND client_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_play_sessions_one_running_per_client;
