// Code generated by sqlc. DO NOT EDIT.

package data

import ()

type Config struct {
	ID         int32
	CreatedAt  interface{}
	UpdatedAt  interface{}
	ServerID   string
	NoAnnounce bool
}

type Entity struct {
	ID        int32
	CreatedAt interface{}
	UpdatedAt interface{}
	Name      string
	ServerID  string
	Karma     int64
}
