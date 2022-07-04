// Package sqlite provides a repository backed by a SQLite database.
package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"

	"github.com/connorkuehl/popple/create"
	poperr "github.com/connorkuehl/popple/errors"
	"github.com/connorkuehl/popple/get"
	"github.com/connorkuehl/popple/repo/sqlite/internal/sqlite"
	"github.com/connorkuehl/popple/update"
)

//go:embed internal/sqlite/schema.sql
var schema string

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) (*Repository, error) {
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	return &Repository{
		db: db,
	}, nil
}

func (r *Repository) CreateEntity(entity create.Entity) error {
	q := sqlite.New(r.db)
	_, err := q.CreateEntity(
		context.Background(),
		sqlite.CreateEntityParams{
			Name:     entity.Name,
			ServerID: entity.ServerID,
		})
	return err
}

func (r *Repository) Entity(serverID, name string) (entity get.Entity, err error) {
	q := sqlite.New(r.db)
	ent, err := q.GetEntity(
		context.Background(),
		sqlite.GetEntityParams{
			ServerID: serverID,
			Name:     name,
		})
	if errors.Is(err, sql.ErrNoRows) {
		err = poperr.ErrNotFound
	}
	if err != nil {
		return get.Entity{}, err
	}

	entity = get.Entity{
		ID:        ent.ID,
		CreatedAt: ent.CreatedAt,
		UpdatedAt: ent.UpdatedAt,
		ServerID:  ent.ServerID,
		Name:      ent.Name,
		Karma:     ent.Karma,
	}
	return entity, err
}

func (r *Repository) Leaderboard(serverID string, limit uint) (board []get.Entity, err error) {
	q := sqlite.New(r.db)
	bd, err := q.GetTopEntities(
		context.Background(),
		sqlite.GetTopEntitiesParams{
			ServerID: serverID,
			Limit:    int32(limit),
		})
	if err != nil {
		return nil, err
	}

	for _, b := range bd {
		entity := get.Entity{
			ID:        b.ID,
			CreatedAt: b.CreatedAt,
			UpdatedAt: b.UpdatedAt,
			ServerID:  b.ServerID,
			Name:      b.Name,
			Karma:     b.Karma,
		}
		board = append(board, entity)
	}

	return board, nil
}

func (r *Repository) Loserboard(serverID string, limit uint) (board []get.Entity, err error) {
	q := sqlite.New(r.db)
	bd, err := q.GetBotEntities(
		context.Background(),
		sqlite.GetBotEntitiesParams{
			ServerID: serverID,
			Limit:    int32(limit),
		})
	if err != nil {
		return nil, err
	}

	for _, b := range bd {
		entity := get.Entity{
			ID:        b.ID,
			CreatedAt: b.CreatedAt,
			UpdatedAt: b.UpdatedAt,
			ServerID:  b.ServerID,
			Name:      b.Name,
			Karma:     b.Karma,
		}
		board = append(board, entity)
	}

	return board, nil
}

func (r *Repository) RemoveEntity(serverID, name string) error {
	q := sqlite.New(r.db)
	return q.RemoveEntity(
		context.Background(),
		sqlite.RemoveEntityParams{
			ServerID: serverID,
			Name:     name,
		})
}

func (r *Repository) UpdateEntity(entity update.Entity) error {
	q := sqlite.New(r.db)
	return q.UpdateEntity(
		context.Background(),
		sqlite.UpdateEntityParams{
			ServerID: entity.ServerID,
			Name:     entity.Name,
			Karma:    entity.Karma,
		})
}

func (r *Repository) Config(serverID string) (get.Config, error) {
	q := sqlite.New(r.db)
	c, err := q.GetConfig(context.Background(), serverID)
	if errors.Is(err, sql.ErrNoRows) {
		err = poperr.ErrNotFound
	}
	if err != nil {
		return get.Config{}, err
	}

	config := get.Config{
		ID:         c.ID,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
		ServerID:   c.ServerID,
		NoAnnounce: c.NoAnnounce,
	}

	return config, nil
}

func (r *Repository) CreateConfig(config create.Config) error {
	q := sqlite.New(r.db)
	_, err := q.CreateConfig(context.Background(), config.ServerID)
	return err
}

func (r *Repository) UpdateConfig(config update.Config) error {
	q := sqlite.New(r.db)
	return q.UpdateConfig(
		context.Background(),
		sqlite.UpdateConfigParams{
			NoAnnounce: config.NoAnnounce,
			ServerID:   config.ServerID,
		})

}
