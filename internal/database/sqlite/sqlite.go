package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"

	_ "modernc.org/sqlite"

	"github.com/connorkuehl/popple/internal/database"
	"github.com/connorkuehl/popple/internal/popple"
)

type Path string

func PathFromEnv() (Path, error) {
	return pathFromEnv(os.Getenv)
}

type DB struct {
	db *sql.DB
}

func New(path Path) (*DB, func(), error) {
	db, err := sql.Open("sqlite", string(path))
	if err != nil {
		return nil, nil, err
	}

	return &DB{db: db}, func() { _ = db.Close() }, nil
}

func (d *DB) Config(ctx context.Context, serverID string) (popple.ServerConfig, error) {
	query := `SELECT server_id, no_announce FROM configs WHERE server_id = $1`
	args := []any{serverID}
	r := d.db.QueryRowContext(ctx, query, args...)

	var c popple.ServerConfig
	err := r.Scan(&c.ServerID, &c.NoAnnounce)
	if errors.Is(err, sql.ErrNoRows) {
		err = database.ErrNotFound
	}
	if err != nil {
		return popple.ServerConfig{}, err
	}
	return c, nil
}

func (d *DB) PutConfig(ctx context.Context, config popple.ServerConfig) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `UPDATE configs SET no_announce = $1, updated_at = datetime('now') WHERE server_id = $2 LIMIT 1`
	args := []any{config.NoAnnounce, config.ServerID}
	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	if affected, _ := res.RowsAffected(); affected == 1 {
		return tx.Commit()
	}

	query = `INSERT INTO configs (
		created_at,
		updated_at,
		server_id,
		no_announce
		) VALUES (datetime('now'), datetime('now'), $1, $2)`
	args = []any{config.ServerID, config.NoAnnounce}
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (d *DB) Entities(ctx context.Context, serverID string, names ...string) ([]popple.Entity, error) {
	var entities []popple.Entity

	// FIXME: This doesn't spark joy, it wasn't obvious how to safely use
	// SELECT foo FROM bar WHERE name IN (?) <-- not sure this is supported
	for _, name := range names {
		query := `SELECT name, karma FROM entities WHERE server_id = $1 AND name = $2 LIMIT 1`
		args := []any{serverID, name}

		var entity popple.Entity
		row := d.db.QueryRowContext(ctx, query, args...)
		err := row.Scan(&entity.Name, &entity.Karma)
		if errors.Is(err, sql.ErrNoRows) {
			entity.Name = name
			err = nil
		}
		if err != nil {
			return nil, err
		}

		entities = append(entities, entity)
	}

	return entities, nil
}

func (d *DB) PutEntities(ctx context.Context, serverID string, entities ...popple.Entity) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	upsert := func(tx *sql.Tx, entity popple.Entity) error {
		query := `UPDATE entities SET karma = $1, updated_at = datetime('now') WHERE name = $2 AND server_id = $3`
		args := []any{entity.Karma, entity.Name, serverID}

		res, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return err
		}

		if affected, _ := res.RowsAffected(); affected == 1 {
			return nil
		}

		query = `INSERT INTO entities (created_at, updated_at, name, server_id, karma) VALUES (datetime('now'), datetime('now'), $1, $2, $3)`
		args = []any{entity.Name, serverID, entity.Karma}

		_, err = tx.ExecContext(ctx, query, args...)
		if err != nil {
			return err
		}

		return nil
	}

	for _, entity := range entities {
		err := upsert(tx, entity)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *DB) Leaderboard(ctx context.Context, serverID string) (popple.Board, error) {
	query := `SELECT name, karma FROM entities WHERE server_id = $1 ORDER BY karma DESC`
	args := []any{serverID}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	var board popple.Board
	for rows.Next() {
		var entry popple.BoardEntry
		if err := rows.Scan(&entry.Who, &entry.Karma); err != nil {
			return nil, err
		}

		board = append(board, entry)
	}

	return board, nil
}

func (d *DB) Loserboard(ctx context.Context, serverID string) (popple.Board, error) {
	query := `SELECT name, karma FROM entities WHERE server_id = $1 ORDER BY karma ASC`
	args := []any{serverID}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	var board popple.Board
	for rows.Next() {
		var entry popple.BoardEntry
		if err := rows.Scan(&entry.Who, &entry.Karma); err != nil {
			return nil, err
		}

		board = append(board, entry)
	}

	return board, nil
}
