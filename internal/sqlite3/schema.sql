CREATE TABLE IF NOT EXISTS entities (
	id INTEGER PRIMARY KEY,
	created_at DATETIME,
	updated_at DATETIME,
	name text NOT NULL,
	server_id TEXT NOT NULL,
	karma BIGINT NOT NULL DEFAULT 0,
	UNIQUE (name, server_id)
);

CREATE TABLE IF NOT EXISTS configs (
	id INTEGER PRIMARY KEY,
	created_at DATETIME,
	updated_at DATETIME,
	server_id TEXT NOT NULL,
	no_announce BOOLEAN NOT NULL DEFAULT FALSE,
	UNIQUE (server_id)
);
