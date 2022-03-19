package create

type Config struct {
	ServerID string
}

type ConfigRepository interface {
	CreateConfig(config Config) error
}
