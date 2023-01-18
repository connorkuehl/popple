// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"github.com/connorkuehl/popple/internal/bot"
	"github.com/connorkuehl/popple/internal/command"
	"github.com/connorkuehl/popple/internal/database/sqlite"
	"github.com/connorkuehl/popple/internal/discord"
	"github.com/google/wire"
)

// Injectors from wire.go:

func InitializeBot() (*bot.Bot, func(), error) {
	token, err := discord.TokenFromEnv()
	if err != nil {
		return nil, nil, err
	}
	dialer := discord.NewDialer(token)
	session, cleanup, err := discord.NewSession(dialer)
	if err != nil {
		return nil, nil, err
	}
	path, err := sqlite.PathFromEnv()
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	db, cleanup2, err := sqlite.New(path)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	router := provideRouter(session)
	botBot := bot.New(session, db, router)
	return botBot, func() {
		cleanup2()
		cleanup()
	}, nil
}

// wire.go:

var DiscordSet = wire.NewSet(discord.NewSession, discord.NewDialer, discord.TokenFromEnv)

var SQLiteSet = wire.NewSet(sqlite.New, sqlite.PathFromEnv)

func provideRouter(s *discord.Session) *command.Router {
	return command.NewRouter("@" + s.Username())
}
