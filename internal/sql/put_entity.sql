UPDATE entities
SET
	updated_at = datetime('now'),
	karma = ?
WHERE name = ? AND server_id = ?
