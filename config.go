package popple

import "github.com/jinzhu/gorm"

// Config represents a server-associated Popple Config.
//
// For example, if someone in Server A tells Popple to stop
// announcing karma events, then it won't affect Popple's
// announce state in Server B.
type Config struct {
	gorm.Model
	GuildID    string
	NoAnnounce bool
}
