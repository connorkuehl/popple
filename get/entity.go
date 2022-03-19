package get

import (
	"time"
)

type Entity struct {
	ID        int64
	CreatedAt time.Time
	UpdatedAt time.Time
	ServerID  string
	Name      string
	Karma     int64
}

type EntityRepository interface {
	Entity(serverID, name string) (Entity, error)
	Leaderboard(serverID string, limit uint) (board []Entity, err error)
	Loserboard(serverID string, limit uint) (board []Entity, err error)
}
