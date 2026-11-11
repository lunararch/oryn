-- name: CreateScanRun :exec
INSERT INTO scan_runs (id, library_id, status, started_at)
VALUES (?, ?, 'running', ?);

-- name: GetScanRun :one
SELECT * FROM scan_runs WHERE id = ?;

-- name: GetCurrentScanRun :one
SELECT * FROM scan_runs WHERE status = 'running' ORDER BY started_at DESC LIMIT 1;

-- name: ListScanRuns :many
SELECT * FROM scan_runs ORDER BY started_at DESC LIMIT ?;

-- name: UpdateScanRunProgress :exec
UPDATE scan_runs SET
    discovered_count = ?,
    added_count = ?,
    updated_count = ?,
    unchanged_count = ?,
    missing_count = ?,
    error_count = ?,
    summary = ?
WHERE id = ?;

-- name: FinishScanRun :exec
UPDATE scan_runs SET
    status = ?,
    finished_at = ?,
    discovered_count = ?,
    added_count = ?,
    updated_count = ?,
    unchanged_count = ?,
    missing_count = ?,
    error_count = ?,
    summary = ?
WHERE id = ?;

-- name: AddScanError :exec
INSERT INTO scan_errors (id, scan_run_id, path, kind, message, created_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: ListScanErrors :many
SELECT * FROM scan_errors WHERE scan_run_id = ? ORDER BY created_at ASC LIMIT ?;

-- name: FailStaleScanRuns :exec
UPDATE scan_runs
SET status = 'failed', finished_at = ?, summary = 'server restarted during scan'
WHERE status = 'running';
