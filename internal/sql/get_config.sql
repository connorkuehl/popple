SELECT
	id,
	created_at,
	updated_at,
	no_announce
FROM configs
WHERE server_id = ?
