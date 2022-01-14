package cli

import "flag"

var (
	DatabasePath = flag.String("database", "popple.sqlite", "Path to the SQLite database file")
	TokenPath    = flag.String("token-file", "popple.token", "Path to the file containing the Discord bot token")
)

func init() {
	flag.Parse()
}
