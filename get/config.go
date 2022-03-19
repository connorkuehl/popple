package get

import (
	"time"
)

type Config struct {
	ID         int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ServerID   string
	NoAnnounce bool
}

type ConfigRepository interface {
	Config(serverID string) (Config, error)
}
