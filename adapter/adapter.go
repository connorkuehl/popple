package adapter

type LeaderboardEntry struct {
	Name  string
	Karma int64
}

type Config struct {
	ServerID   string
	NoAnnounce bool
}

type Entity struct {
	ServerID string
	Name     string
	Karma    int64
}

type PersistenceLayer interface {
	CreateConfig(serverID string) error
	CreateEntity(serverID, name string) error
	GetConfig(serverID string) (Config, error)
	GetEntity(serverID, name string) (Entity, error)
	GetBotEntities(serverID string, limit uint) ([]Entity, error)
	GetTopEntities(serverID string, limit uint) ([]Entity, error)
	PutConfig(cfg Config) error
	AddKarmaToEntity(entity Entity, karma int64) (Entity, error)
}
