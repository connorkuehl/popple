package adapter

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	_ "modernc.org/sqlite"

	"github.com/connorkuehl/popple/internal/sqlite3"
	"github.com/connorkuehl/popple/internal/sqlite3/data"
)

type Sqlite struct {
	mu sync.Mutex
	db *sql.DB
}

func NewSQLitePersistenceLayer(db *sql.DB) (*Sqlite, error) {
	_, err := db.Exec(sqlite3.SchemaSQL)
	if err != nil {
		return nil, err
	}

	s := Sqlite{
		db: db,
		mu: sync.Mutex{},
	}

	return &s, nil
}

func (s *Sqlite) CreateConfig(serverID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := data.New(s.db).CreateConfig(context.Background(), serverID)
	return err
}

func (s *Sqlite) CreateEntity(serverID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := data.New(s.db).CreateEntity(context.Background(), data.CreateEntityParams{
		ServerID: serverID,
		Name:     name,
	})
	return err
}

func (s *Sqlite) GetConfig(serverID string) (Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := data.New(s.db).GetConfig(context.Background(), serverID)
	if err != nil {
		return Config{}, err
	}

	c := Config{
		ServerID:   cfg.ServerID,
		NoAnnounce: cfg.NoAnnounce,
	}

	return c, err
}

func (s *Sqlite) GetEntity(serverID, name string) (Entity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entity, err := data.New(s.db).GetEntity(context.Background(), data.GetEntityParams{
		ServerID: serverID,
		Name:     name,
	})
	if err != nil {
		return Entity{}, err
	}

	e := Entity{
		entity.Name,
		entity.ServerID,
		entity.Karma,
	}

	return e, err
}

func (s *Sqlite) GetBotEntities(serverID string, limit uint) ([]Entity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var entities []Entity

	es, err := data.New(s.db).GetBotEntities(context.Background(), data.GetBotEntitiesParams{
		ServerID: serverID,
		Limit:    int32(limit),
	})
	if err != nil {
		return entities, err
	}

	entities = make([]Entity, 0, len(es))
	for _, e := range es {
		entities = append(entities, Entity{
			ServerID: e.ServerID,
			Name:     e.Name,
			Karma:    e.Karma,
		})
	}

	return entities, err
}

func (s *Sqlite) GetTopEntities(serverID string, limit uint) ([]Entity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var entities []Entity

	es, err := data.New(s.db).GetTopEntities(context.Background(), data.GetTopEntitiesParams{
		ServerID: serverID,
		Limit:    int32(limit),
	})
	if err != nil {
		return entities, err
	}

	entities = make([]Entity, 0, len(es))
	for _, e := range es {
		entities = append(entities, Entity{
			ServerID: e.ServerID,
			Name:     e.Name,
			Karma:    e.Karma,
		})
	}

	return entities, err
}

func (s *Sqlite) PutConfig(cfg Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return data.New(s.db).PutConfig(context.Background(), data.PutConfigParams{
		ServerID:   cfg.ServerID,
		NoAnnounce: cfg.NoAnnounce,
	})
}

func (s *Sqlite) AddKarmaToEntity(entity Entity, karma int64) (Entity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return entity, err
	}
	defer func() { _ = tx.Rollback() }()

	qs := data.New(s.db).WithTx(tx)

	prsEnt, err := qs.GetEntity(context.Background(), data.GetEntityParams{
		ServerID: entity.ServerID,
		Name:     entity.Name,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Entity{}, err
	}

	entity = Entity{
		ServerID: prsEnt.ServerID,
		Name:     prsEnt.Name,
		Karma:    prsEnt.Karma + int64(karma),
	}

	if entity.Karma == 0 {
		err = qs.DeleteEntity(context.Background(), prsEnt.ID)
	} else {
		err = qs.PutEntity(context.Background(), data.PutEntityParams{
			ID:    prsEnt.ID,
			Karma: entity.Karma,
		})
	}
	if err != nil {
		return Entity{}, err
	}

	_ = tx.Commit()
	return entity, err
}
