-- name: CreateCollection :exec
INSERT INTO collections (id, name, description, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetCollection :one
SELECT c.*,
       (SELECT COUNT(*) FROM collection_games cg WHERE cg.collection_id = c.id) AS game_count
FROM collections c WHERE c.id = ?;

-- name: ListCollections :many
SELECT c.*,
       (SELECT COUNT(*) FROM collection_games cg WHERE cg.collection_id = c.id) AS game_count
FROM collections c
ORDER BY c.name COLLATE NOCASE ASC;

-- name: ListCollectionsForGame :many
SELECT c.*,
       (SELECT COUNT(*) FROM collection_games cg2 WHERE cg2.collection_id = c.id) AS game_count,
       EXISTS(SELECT 1 FROM collection_games cg WHERE cg.collection_id = c.id AND cg.game_id = ?) AS contains
FROM collections c
ORDER BY c.name COLLATE NOCASE ASC;

-- name: UpdateCollection :exec
UPDATE collections SET name = ?, description = ?, updated_at = ? WHERE id = ?;

-- name: DeleteCollection :exec
DELETE FROM collections WHERE id = ?;

-- name: AddGameToCollection :exec
INSERT INTO collection_games (collection_id, game_id, added_at)
VALUES (?, ?, ?)
ON CONFLICT(collection_id, game_id) DO NOTHING;

-- name: RemoveGameFromCollection :exec
DELETE FROM collection_games WHERE collection_id = ? AND game_id = ?;

-- name: GetCollectionName :one
SELECT name FROM collections WHERE id = ?;
