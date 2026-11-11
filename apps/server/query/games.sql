-- name: InsertGame :exec
INSERT INTO games (
    id, library_id, relative_path, folder_name, detected_title,
    detected_description, detected_developer, detected_publisher,
    detected_release_date, detected_version, detection_confidence,
    detection_evidence, availability_status, filesystem_fingerprint,
    first_seen_at, last_seen_at, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: UpdateGameDetected :exec
UPDATE games SET
    folder_name = ?,
    detected_title = ?,
    detected_description = ?,
    detected_developer = ?,
    detected_publisher = ?,
    detected_release_date = ?,
    detected_version = ?,
    detection_confidence = ?,
    detection_evidence = ?,
    availability_status = 'available',
    filesystem_fingerprint = ?,
    last_seen_at = ?,
    updated_at = ?
WHERE id = ?;

-- name: UpdateGamePath :exec
UPDATE games SET
    relative_path = ?,
    folder_name = ?,
    availability_status = 'available',
    last_seen_at = ?,
    updated_at = ?
WHERE id = ?;

-- name: SetGameAvailability :exec
UPDATE games SET availability_status = ?, last_seen_at = ?, updated_at = ? WHERE id = ?;

-- name: ListGamesForLibrary :many
SELECT id, relative_path, folder_name, detected_title, availability_status,
       filesystem_fingerprint, first_seen_at, last_seen_at
FROM games
WHERE library_id = ?
ORDER BY relative_path ASC;

-- name: GetGame :one
SELECT g.*,
       o.title AS override_title,
       o.description AS override_description,
       o.developer AS override_developer,
       o.publisher AS override_publisher,
       o.release_date AS override_release_date,
       o.rating AS override_rating,
       o.notes AS override_notes,
       o.completion_status AS override_completion_status,
       EXISTS(SELECT 1 FROM favorites f WHERE f.game_id = g.id) AS is_favorite,
       CAST(COALESCE((SELECT a.id FROM artwork a WHERE a.game_id = g.id AND a.kind = 'cover'), '') AS TEXT) AS cover_artwork_id,
       CAST(COALESCE((SELECT a.id FROM artwork a WHERE a.game_id = g.id AND a.kind = 'banner'), '') AS TEXT) AS banner_artwork_id
FROM games g
LEFT JOIN game_overrides o ON o.game_id = g.id
WHERE g.id = ?;

-- name: GetGameIDByPath :one
SELECT id FROM games WHERE library_id = ? AND relative_path = ?;

-- name: GetGameIDByFingerprint :many
SELECT id FROM games
WHERE library_id = ? AND filesystem_fingerprint = ? AND availability_status = 'missing' AND filesystem_fingerprint <> '';

-- name: CountGames :one
SELECT COUNT(*) FROM games WHERE library_id = ?;

-- name: CountAvailableGames :one
SELECT COUNT(*) FROM games WHERE library_id = ? AND availability_status = 'available';

-- name: SetFavorite :exec
INSERT INTO favorites (game_id, created_at) VALUES (?, ?)
ON CONFLICT(game_id) DO NOTHING;

-- name: DeleteFavorite :exec
DELETE FROM favorites WHERE game_id = ?;

-- name: GetGameOverride :one
SELECT * FROM game_overrides WHERE game_id = ?;

-- name: UpsertGameOverride :exec
INSERT INTO game_overrides (
    game_id, title, description, developer, publisher, release_date,
    rating, notes, completion_status, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(game_id) DO UPDATE SET
    title = excluded.title,
    description = excluded.description,
    developer = excluded.developer,
    publisher = excluded.publisher,
    release_date = excluded.release_date,
    rating = excluded.rating,
    notes = excluded.notes,
    completion_status = excluded.completion_status,
    updated_at = excluded.updated_at;

-- name: DeleteGameOverride :exec
DELETE FROM game_overrides WHERE game_id = ?;

-- name: GetStats :one
SELECT
    COUNT(*) AS total_games,
    COUNT(*) FILTER (WHERE availability_status = 'available') AS available_games,
    COUNT(*) FILTER (WHERE availability_status = 'missing') AS missing_games,
    (SELECT COUNT(*) FROM favorites) AS favorite_games
FROM games
WHERE library_id = ?;

-- name: UpsertArtwork :exec
INSERT INTO artwork (
    id, game_id, kind, origin, source_rel_path, cache_key,
    mime_type, width, height, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(game_id, kind) DO UPDATE SET
    origin = excluded.origin,
    source_rel_path = excluded.source_rel_path,
    cache_key = excluded.cache_key,
    mime_type = excluded.mime_type,
    width = excluded.width,
    height = excluded.height,
    updated_at = excluded.updated_at;

-- name: GetArtwork :one
SELECT * FROM artwork WHERE id = ?;

-- name: GetArtworkForGame :one
SELECT * FROM artwork WHERE game_id = ? AND kind = ?;

-- name: ListArtworkForGame :many
SELECT * FROM artwork WHERE game_id = ?;

-- name: DeleteArtwork :exec
DELETE FROM artwork WHERE game_id = ? AND kind = ?;
