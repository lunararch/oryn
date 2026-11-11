-- name: CreateDesktopClient :exec
INSERT INTO desktop_clients (id, name, platform, token_hash, registered_at, last_seen_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetDesktopClientByTokenHash :one
SELECT * FROM desktop_clients WHERE token_hash = ? LIMIT 1;

-- name: GetDesktopClient :one
SELECT * FROM desktop_clients WHERE id = ? LIMIT 1;

-- name: ListDesktopClients :many
SELECT * FROM desktop_clients ORDER BY registered_at DESC;

-- name: TouchDesktopClient :exec
UPDATE desktop_clients SET last_seen_at = ? WHERE id = ?;

-- name: RevokeDesktopClient :exec
UPDATE desktop_clients SET revoked_at = ? WHERE id = ?;

-- name: CreatePairingCode :exec
INSERT INTO pairing_codes (id, code_hash, expires_at, created_at)
VALUES (?, ?, ?, ?);

-- name: ListActivePairingCodes :many
SELECT * FROM pairing_codes
WHERE used_at IS NULL AND expires_at > ?
ORDER BY created_at DESC;

-- name: MarkPairingCodeUsed :exec
UPDATE pairing_codes SET used_at = ? WHERE id = ?;

-- name: DeleteExpiredPairingCodes :exec
DELETE FROM pairing_codes WHERE expires_at <= ? OR used_at IS NOT NULL;

-- name: InsertDesktopSession :exec
INSERT INTO play_sessions (
    id, game_id, client_id, started_at, ended_at, duration_seconds,
    status, source, exit_code, notes, created_at, updated_at, last_heartbeat_at
) VALUES (?, ?, ?, ?, NULL, 0, 'running', 'desktop', NULL, '', ?, ?, ?);

-- name: GetRunningSessionForClient :one
SELECT * FROM play_sessions
WHERE client_id = ? AND status = 'running'
ORDER BY started_at DESC LIMIT 1;

-- name: HeartbeatSession :exec
UPDATE play_sessions SET last_heartbeat_at = ?, updated_at = ? WHERE id = ?;

-- name: FinishSession :exec
UPDATE play_sessions SET
    status = ?,
    ended_at = ?,
    duration_seconds = ?,
    exit_code = ?,
    updated_at = ?
WHERE id = ?;

-- name: ListStaleRunningSessions :many
SELECT * FROM play_sessions
WHERE status = 'running' AND COALESCE(last_heartbeat_at, started_at) < ?;

-- name: MarkSessionInterrupted :exec
UPDATE play_sessions SET
    status = 'interrupted',
    ended_at = ?,
    duration_seconds = ?,
    updated_at = ?
WHERE id = ?;
