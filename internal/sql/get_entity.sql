SELECT
	id,
	created_at,
	updated_at,
	karma
FROM entities
WHERE NAME = ? AND server_id = ?
