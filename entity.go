package popple

import "github.com/jinzhu/gorm"

// Entity represents a karma subject in the persistence layer.
type Entity struct {
	gorm.Model
	Name    string
	GuildID string
	Karma   int
}
