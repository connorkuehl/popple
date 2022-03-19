package null

import (
	"sort"
	"time"

	"github.com/connorkuehl/popple/create"
	"github.com/connorkuehl/popple/errors"
	"github.com/connorkuehl/popple/get"
	"github.com/connorkuehl/popple/update"
)

type Entity struct {
	ID        int64
	ServerID  string
	Name      string
	Karma     int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Config struct {
	ID         int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ServerID   string
	NoAnnounce bool
}

type Repository struct {
	nextID   int64
	entities map[string]map[string]Entity
	configs  map[string]Config
}

func NewRepository() *Repository {
	return &Repository{
		nextID:   1,
		entities: make(map[string]map[string]Entity),
		configs:  make(map[string]Config),
	}
}

func (n *Repository) CreateEntity(entity create.Entity) error {
	srv, ok := n.entities[entity.ServerID]
	if !ok {
		n.entities[entity.ServerID] = make(map[string]Entity)
		srv = n.entities[entity.ServerID]
	}

	_, ok = srv[entity.Name]
	if ok {
		return errors.ErrNotFound
	}

	t := time.Now()

	srv[entity.Name] = Entity{
		ID:        n.nextID,
		CreatedAt: t,
		UpdatedAt: t,
		ServerID:  entity.ServerID,
		Name:      entity.Name,
	}

	n.nextID++

	return nil
}

func (n *Repository) Entity(serverID, name string) (entity get.Entity, err error) {
	srv, ok := n.entities[serverID]
	if !ok {
		return get.Entity{}, errors.ErrNotFound
	}

	e, ok := srv[name]
	if !ok {
		return get.Entity{}, errors.ErrNotFound
	}

	entity = get.Entity{
		ID:        e.ID,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
		ServerID:  e.ServerID,
		Name:      e.Name,
		Karma:     e.Karma,
	}
	return entity, nil
}

func (n *Repository) Leaderboard(serverID string, limit uint) (board []get.Entity, err error) {
	if uint(len(n.entities)) < limit {
		limit = uint(len(n.entities))
	}

	for _, data := range n.entities[serverID] {
		e := get.Entity{
			Name:      data.Name,
			Karma:     data.Karma,
			CreatedAt: data.CreatedAt,
			UpdatedAt: data.UpdatedAt,
		}
		board = append(board, e)
	}

	sort.Slice(board, func(i, j int) bool {
		return board[i].Karma < board[j].Karma
	})

	return board[:limit], nil
}

func (n *Repository) Loserboard(serverID string, limit uint) (board []get.Entity, err error) {
	if uint(len(n.entities)) < limit {
		limit = uint(len(n.entities))
	}

	for _, data := range n.entities[serverID] {
		e := get.Entity{
			Name:      data.Name,
			Karma:     data.Karma,
			CreatedAt: data.CreatedAt,
			UpdatedAt: data.UpdatedAt,
		}
		board = append(board, e)
	}

	sort.Slice(board, func(i, j int) bool {
		return board[i].Karma > board[j].Karma
	})

	return board[:limit], nil
}

func (n *Repository) RemoveEntity(serverID, name string) error {
	srv, ok := n.entities[serverID]
	if !ok {
		return errors.ErrNotFound
	}

	_, ok = srv[name]
	if !ok {
		return errors.ErrNotFound
	}

	delete(srv, name)
	return nil
}

func (n *Repository) UpdateEntity(entity update.Entity) error {
	srv, ok := n.entities[entity.ServerID]
	if !ok {
		return errors.ErrNotFound
	}
	temp, ok := srv[entity.Name]
	if !ok {
		return errors.ErrNotFound
	}

	temp.UpdatedAt = time.Now()
	temp.Karma = entity.Karma
	n.entities[entity.ServerID][entity.Name] = temp

	return nil
}

func (n *Repository) Config(serverID string) (get.Config, error) {
	config, ok := n.configs[serverID]
	if !ok {
		return get.Config{}, errors.ErrNotFound
	}

	c := get.Config{
		ID:         config.ID,
		CreatedAt:  config.CreatedAt,
		UpdatedAt:  config.UpdatedAt,
		ServerID:   config.ServerID,
		NoAnnounce: config.NoAnnounce,
	}
	return c, nil
}

func (n *Repository) CreateConfig(config create.Config) error {
	_, ok := n.configs[config.ServerID]
	if ok {
		return errors.ErrAlreadyExists
	}

	n.configs[config.ServerID] = Config{ServerID: config.ServerID}
	return nil
}

func (n *Repository) UpdateConfig(config update.Config) error {
	c, ok := n.configs[config.ServerID]
	if !ok {
		return errors.ErrNotFound
	}

	c.NoAnnounce = config.NoAnnounce
	n.configs[config.ServerID] = c
	return nil
}
