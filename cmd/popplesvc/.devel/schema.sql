CREATE TABLE IF NOT EXISTS entities (
  id 		        INT AUTO_INCREMENT NOT NULL,
  created_at 	  DATETIME NOT NULL,
  updated_at 	  DATETIME NOT NULL,
  name 		      VARCHAR(128) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  server_id 	  VARCHAR(128) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  karma BIGINT 	DEFAULT 0,
  PRIMARY KEY   (`id`)
);

CREATE TABLE IF NOT EXISTS configs (
  id 		INT AUTO_INCREMENT NOT NULL,
  created_at 	DATETIME NOT NULL,
  updated_at 	DATETIME NOT NULL,
  server_id 	VARCHAR(128) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,
  no_announce	BOOLEAN DEFAULT false,
  PRIMARY KEY (`id`)
);
