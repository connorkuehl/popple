package main

import "github.com/jinzhu/gorm"

// config represents a server-associated Popple config.
//
// For example, if someone in Server A tells Popple to stop
// announcing karma events, then it won't affect Popple's
// announce state in Server B.
type config struct {
	gorm.Model
	guildID    string
	noAnnounce bool
}
