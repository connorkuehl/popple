UPDATE configs
SET
	updated_at = datetime('now'),
	no_announce = ?
WHERE server_id = ?
