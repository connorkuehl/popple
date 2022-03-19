package remove

type EntityRepository interface {
	RemoveEntity(serverID, name string) error
}
