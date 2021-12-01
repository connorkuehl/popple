SELECT
	id,
	created_at,
	updated_at,
	name,
	server_id,
	karma
FROM entities
WHERE server_id = ?
ORDER BY karma DESC LIMIT ?
