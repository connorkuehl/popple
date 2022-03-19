package create

type Entity struct {
	Name     string
	ServerID string
}

type EntityRepository interface {
	CreateEntity(entity Entity) error
}
