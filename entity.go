package main

import "github.com/jinzhu/gorm"

// entity represents a karma subject in the persistence layer.
type entity struct {
	gorm.Model
	Name    string
	GuildID string
	Karma   int
}
