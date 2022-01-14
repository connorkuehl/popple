package sqlite3

import (
	_ "embed"
)

//go:embed schema.sql
var SchemaSQL string
