CREATE TABLE IF NOT EXISTS entities (
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  name VARCHAR(128) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  server_id VARCHAR(128) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  karma BIGINT DEFAULT 0,
  UNIQUE (server_id, name)
);

CREATE TABLE IF NOT EXISTS configs (
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  server_id VARCHAR(128) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  no_announce BOOLEAN DEFAULT false,
  PRIMARY KEY (server_id)
);
