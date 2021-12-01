CREATE TABLE IF NOT EXISTS `entities` (
	`id` INTEGER PRIMARY KEY ASC AUTOINCREMENT,
	`created_at` DATETIME,
	`updated_at` DATETIME,
	`name` text NOT NULL,
	`server_id` TEXT NOT NULL,
	`karma` INTEGER DEFAULT 0,
	UNIQUE (name, server_id)
);

CREATE TABLE IF NOT EXISTS `configs` (
	`id` INTEGER PRIMARY KEY ASC AUTOINCREMENT,
	`created_at` DATETIME,
	`updated_at` DATETIME,
	`server_id` TEXT NOT NULL,
	`no_announce` BOOLEAN DEFAULT FALSE,
	UNIQUE (server_id)
);
