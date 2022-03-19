package update

type Config struct {
	ServerID   string
	NoAnnounce bool
}

type ConfigRepository interface {
	UpdateConfig(config Config) error
}
