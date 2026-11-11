-- name: CreateLibrary :exec
INSERT INTO libraries (id, name, root_path, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetLibraryByRoot :one
SELECT * FROM libraries WHERE root_path = ? LIMIT 1;

-- name: GetLibrary :one
SELECT * FROM libraries WHERE id = ? LIMIT 1;

-- name: ListLibraries :many
SELECT * FROM libraries ORDER BY name COLLATE NOCASE ASC;

-- name: TouchLibraryScan :exec
UPDATE libraries SET last_scan_at = ?, updated_at = ? WHERE id = ?;
