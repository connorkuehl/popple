package sqlite

import (
	"database/sql"
	_ "embed"

	_ "modernc.org/sqlite"
)

//go:embed migrations/000001_create_tables.up.sql
var up string

func NewInMemory() (*DB, func(), error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, func() {}, err
	}

	_, err = db.Exec(up)
	if err != nil {
		db.Close()
		return nil, func() {}, err
	}

	return &DB{db}, func() { db.Close() }, nil
}
