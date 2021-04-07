package main

import "github.com/jinzhu/gorm"

type Config struct {
	gorm.Model
	GuildID string
}
