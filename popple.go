package popple

type Increments map[string]int64

type Config struct {
	ServerID   string `json:"server_id"`
	NoAnnounce bool   `json:"no_announce"`
}

type BoardEntry struct {
	Who   string `json:"who"`
	Karma int64  `json:"karma"`
}

type Board []BoardEntry

type BoardOrder int

const (
	BoardOrderAsc BoardOrder = 1
	BoardOrderDsc BoardOrder = 2
)
