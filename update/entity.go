package update

type Entity struct {
	ServerID string
	Name     string
	Karma    int64
}

type EntityRepository interface {
	UpdateEntity(entity Entity) error
}
