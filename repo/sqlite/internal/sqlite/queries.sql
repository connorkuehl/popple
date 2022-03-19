-- name: GetEntity :one
SELECT * FROM entities WHERE server_id = $1 AND name = $2;

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


-- name: RemoveEntity :exec
DELETE FROM entities WHERE server_id = $1 AND name = $2;

-- name: UpdateEntity :exec
UPDATE entities SET
    updated_at = datetime('now'),
    karma = $3
WHERE server_id = $1 AND name = $2;

-- name: GetTopEntities :many
SELECT * FROM entities
WHERE server_id = $1 ORDER BY karma DESC LIMIT $2;

-- name: GetBotEntities :many
SELECT * FROM entities
WHERE server_id = $1 ORDER BY karma ASC LIMIT $2;

-- name: GetConfig :one
SELECT * FROM configs WHERE server_id = $1;

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

-- name: UpdateConfig :exec
UPDATE configs SET
    updated_at = datetime('now'),
    no_announce = $1
WHERE server_id = $2;
