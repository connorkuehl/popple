package data

import (
	"database/sql"
	"time"

	query "github.com/connorkuehl/popple/internal/sql"
)

type Entity struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string
	ServerID  string
	Karma     int
}

type Config struct {
	ID         uint
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ServerID   string
	NoAnnounce bool
}

func MakeTables(db *sql.DB) error {
	_, err := db.Exec(query.MakeTables)
	return err
}

func GetConfig(db *sql.DB, config *Config) error {
	row := db.QueryRow(query.GetConfig, config.ServerID)
	err := row.Err()
	if err != nil {
		return err
	}

	err = row.Scan(
		&config.ID,
		&config.CreatedAt,
		&config.UpdatedAt,
		&config.NoAnnounce,
	)
	return err
}

func PutConfig(db *sql.DB, config Config) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, _ = tx.Exec(query.CreateConfig, config.ServerID)
	_, err = tx.Exec(query.PutConfig, config.NoAnnounce, config.ServerID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	err = tx.Commit()
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return nil
}

func DeleteEntity(db *sql.DB, entity Entity) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(query.DeleteEntity, entity.Name, entity.ServerID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return nil
}

func GetEntity(db *sql.DB, entity *Entity) error {
	row := db.QueryRow(query.GetEntity, entity.Name, entity.ServerID)
	err := row.Err()
	if err != nil {
		return err
	}

	err = row.Scan(
		&entity.ID,
		&entity.CreatedAt,
		&entity.UpdatedAt,
		&entity.Karma,
	)

	return err
}

type Sort int

const (
	Ascending Sort = iota
	Descending
)

func GetTopEntities(db *sql.DB, serverID string, limit uint) ([]Entity, error) {
	return getEntities(db, query.GetTopEntities, serverID, limit)
}

func GetBotEntities(db *sql.DB, serverID string, limit uint) ([]Entity, error) {
	return getEntities(db, query.GetBotEntities, serverID, limit)
}

func getEntities(db *sql.DB, theQuery string, serverID string, limit uint) ([]Entity, error) {
	var entities []Entity

	rows, err := db.Query(theQuery, serverID, limit)
	if err != nil {
		return entities, err
	}

	for rows.Next() {
		var e Entity

		err := rows.Scan(
			&e.ID,
			&e.CreatedAt,
			&e.UpdatedAt,
			&e.Name,
			&e.ServerID,
			&e.Karma,
		)
		if err != nil {
			return entities, err
		}

		entities = append(entities, e)
	}

	return entities, nil
}

func PutEntity(db *sql.DB, entity Entity) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, _ = tx.Exec(query.CreateEntity, entity.Name, entity.ServerID)
	_, err = tx.Exec(query.PutEntity, entity.Karma, entity.Name, entity.ServerID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return nil
}
