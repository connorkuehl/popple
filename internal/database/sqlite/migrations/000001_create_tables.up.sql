CREATE TABLE IF NOT EXISTS entities (
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name TEXT NOT NULL,
    server_id TEXT NOT NULL,
    karma BIGINT NOT NULL DEFAULT 0,
    UNIQUE (name, server_id)
);

CREATE TABLE IF NOT EXISTS configs (
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    server_id TEXT NOT NULL,
	no_announce BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (server_id)
);