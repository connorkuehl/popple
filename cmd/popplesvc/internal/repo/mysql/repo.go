// Package mysql is a Popple persistence layer adapter to a MySQL
// database.
package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/connorkuehl/popple"
	poperr "github.com/connorkuehl/popple/errors"
)

// Repository is a MySQL repository.
type Repository struct {
	db *sql.DB
}

// New creates a new MySQL repository.
func New(db *sql.DB) *Repository {
	return &Repository{db}
}

// CreateConfig creates a new config.
func (r *Repository) CreateConfig(c popple.Config) error {
	_, err := r.db.ExecContext(context.TODO(),
		`INSERT INTO configs (created_at, updated_at, server_id, no_announce) VALUES (NOW(), NOW(), ?, ?)`,
		c.ServerID, false,
	)
	return err
}

// Config fetches a config by its server ID.
func (r *Repository) Config(serverID string) (popple.Config, error) {
	row := r.db.QueryRowContext(context.TODO(),
		`SELECT id, created_at, updated_at, no_announce FROM configs WHERE server_id=?`,
		serverID,
	)
	c := popple.Config{ServerID: serverID}
	err := row.Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt, &c.NoAnnounce)
	if errors.Is(err, sql.ErrNoRows) {
		err = poperr.ErrNotFound
	}
	if err != nil {
		return popple.Config{}, err
	}

	return c, nil
}

// CreateEntity creates an entity.
func (r *Repository) CreateEntity(entity popple.Entity) error {
	_, err := r.db.ExecContext(context.TODO(),
		`INSERT INTO entities (created_at, updated_at, name, server_id, karma) VALUES (NOW(), NOW(), ?, ?, ?)`,
		entity.Name, entity.ServerID, 0,
	)
	if err != nil {
		return err
	}
	return nil
}

// UpdateConfig updates a config.
func (r *Repository) UpdateConfig(config popple.Config) error {
	_, err := r.db.ExecContext(context.TODO(),
		`UPDATE configs SET updated_at=NOW(), no_announce=? WHERE server_id=? LIMIT 1`,
		config.NoAnnounce, config.ServerID,
	)
	return err
}

// Entity fetches an entity by its name and server ID.
func (r *Repository) Entity(serverID, name string) (popple.Entity, error) {
	row := r.db.QueryRowContext(context.TODO(),
		`SELECT id, created_at, updated_at, karma FROM entities WHERE server_id=? AND name=?`,
		serverID, name,
	)
	e := popple.Entity{ServerID: serverID, Name: name}
	err := row.Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt, &e.Karma)
	if errors.Is(err, sql.ErrNoRows) {
		err = poperr.ErrNotFound
	}
	if err != nil {
		return popple.Entity{}, err
	}
	if err != nil {
		return popple.Entity{}, err
	}

	return e, nil
}

// UpdateEntity updates an entity.
func (r *Repository) UpdateEntity(entity popple.Entity) error {
	_, err := r.db.ExecContext(context.TODO(),
		`UPDATE entities SET updated_at=NOW(), karma=? WHERE server_id=? AND name=? LIMIT 1`,
		entity.Karma, entity.ServerID, entity.Name,
	)
	return err
}

// RemoveEntity removes an entity.
func (r *Repository) RemoveEntity(serverID, name string) error {
	_, err := r.db.ExecContext(context.TODO(),
		`DELETE FROM entities WHERE server_id=? AND name=? LIMIT 1`,
		serverID, name,
	)
	return err
}

// Leaderboard fetches the top-N entities with the most karma.
func (r *Repository) Leaderboard(serverID string, limit uint) ([]popple.Entity, error) {
	var board []popple.Entity
	rows, err := r.db.QueryContext(context.TODO(),
		`SELECT id, created_at, updated_at, name, karma FROM entities WHERE server_id=? ORDER BY karma DESC LIMIT ?`,
		serverID, limit,
	)
	if err != nil {
		return board, err
	}
	defer rows.Close()

	for rows.Next() {
		e := popple.Entity{ServerID: serverID}
		err := rows.Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt, &e.Name, &e.Karma)
		if err != nil {
			return board, err
		}
		board = append(board, e)
	}
	return board, rows.Err()
}

// Loserboard fetches the bottom-N entities with the least amount of karma.
func (r *Repository) Loserboard(serverID string, limit uint) ([]popple.Entity, error) {
	var board []popple.Entity
	rows, err := r.db.QueryContext(context.TODO(),
		`SELECT id, created_at, updated_at, name, karma FROM entities WHERE server_id=? ORDER BY karma ASC LIMIT ?`,
		serverID, limit,
	)
	if err != nil {
		return board, err
	}
	defer rows.Close()

	for rows.Next() {
		e := popple.Entity{ServerID: serverID}
		err := rows.Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt, &e.Name, &e.Karma)
		if err != nil {
			return board, err
		}
		board = append(board, e)
	}
	return board, rows.Err()
}
