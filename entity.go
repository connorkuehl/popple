package main

import "github.com/jinzhu/gorm"

type Entity struct {
	gorm.Model
	Name    string
	GuildID string
	Karma   int
}
