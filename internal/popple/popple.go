package popple

type Increments map[string]int64

type Entity struct {
	Name  string
	Karma int64
}

type ServerConfig struct {
	ServerID   string
	NoAnnounce bool
}

type BoardEntry struct {
	Who   string
	Karma int64
}

type Board []BoardEntry

type BoardOrder int

const (
	BoardOrderAsc BoardOrder = 1
	BoardOrderDsc BoardOrder = 2
)
