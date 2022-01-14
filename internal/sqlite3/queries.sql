-- name: CreateConfig :execresult
INSERT INTO configs (
    created_at,
    updated_at,
    server_id
) VALUES (
    datetime('now'),
    datetime('now'),
    $1
);

-- name: DeleteConfig :exec
DELETE FROM configs WHERE id = $1;

-- name: GetConfig :one
SELECT * FROM configs
WHERE server_id = $1;

-- name: PutConfig :exec
UPDATE configs SET
    updated_at = datetime('now'),
    no_announce = $1
WHERE server_id = $2;

-- name: CreateEntity :execresult
INSERT INTO entities (
    created_at,
    updated_at,
    name,
    server_id
) VALUES (
    datetime('now'),
    datetime('now'),
    $1,
    $2
);

-- name: DeleteEntity :exec
DELETE FROM entities WHERE id = $1;

-- name: GetEntity :one
SELECT * FROM entities
WHERE server_id = $1 AND name = $2;

-- name: GetTopEntities :many
SELECT * FROM entities
WHERE server_id = $1 ORDER BY karma DESC LIMIT $2;

-- name: GetBotEntities :many
SELECT * FROM entities
WHERE server_id = $1 ORDER BY karma ASC LIMIT $2;

-- name: PutEntity :exec
UPDATE entities SET
    updated_at = datetime('now'),
    karma = $1
WHERE id = $2;
