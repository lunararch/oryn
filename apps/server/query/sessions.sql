-- name: InsertPlaySession :exec
INSERT INTO play_sessions (
    id, game_id, client_id, started_at, ended_at, duration_seconds,
    status, source, exit_code, notes, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetPlaySession :one
SELECT * FROM play_sessions WHERE id = ?;

-- name: ListPlaySessionsForGame :many
SELECT * FROM play_sessions
WHERE game_id = ?
ORDER BY started_at DESC
LIMIT ? OFFSET ?;

-- name: CountPlaySessionsForGame :one
SELECT COUNT(*) FROM play_sessions WHERE game_id = ?;

-- name: ListRecentPlaySessions :many
SELECT * FROM play_sessions
ORDER BY started_at DESC
LIMIT ?;

-- name: UpdatePlaySession :exec
UPDATE play_sessions SET
    started_at = ?,
    ended_at = ?,
    duration_seconds = ?,
    status = ?,
    notes = ?,
    updated_at = ?
WHERE id = ?;

-- name: DeletePlaySession :exec
DELETE FROM play_sessions WHERE id = ?;

-- name: GetGamePlaytime :one
SELECT COALESCE(SUM(duration_seconds), 0) AS seconds
FROM play_sessions
WHERE game_id = ? AND status IN ('completed', 'interrupted');

-- name: GetPlayStats :one
SELECT
    COUNT(DISTINCT ps.game_id) AS played_games,
    CAST(COALESCE(SUM(ps.duration_seconds), 0) AS INTEGER) AS total_seconds
FROM play_sessions ps
JOIN games g ON g.id = ps.game_id
WHERE g.library_id = ? AND ps.status IN ('completed', 'interrupted');
