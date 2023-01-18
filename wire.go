//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"

	"github.com/connorkuehl/popple/internal/bot"
	"github.com/connorkuehl/popple/internal/command"
	"github.com/connorkuehl/popple/internal/database/sqlite"
	"github.com/connorkuehl/popple/internal/discord"
)

var DiscordSet = wire.NewSet(
	discord.NewSession,
	discord.NewDialer,
	discord.TokenFromEnv,
)

var SQLiteSet = wire.NewSet(
	sqlite.New,
	sqlite.PathFromEnv,
)

func provideRouter(s *discord.Session) *command.Router {
	return command.NewRouter("@" + s.Username())
}

func InitializeBot() (*bot.Bot, func(), error) {
	wire.Build(
		bot.New,
		provideRouter,
		wire.Bind(new(bot.CommandRouter), new(*command.Router)),
		wire.Bind(new(bot.Session), new(*discord.Session)),
		DiscordSet,
		wire.Bind(new(bot.DB), new(*sqlite.DB)),
		SQLiteSet,
	)
	return nil, nil, nil
}
