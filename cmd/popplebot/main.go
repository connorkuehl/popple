package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/connorkuehl/popple"
	"github.com/connorkuehl/popple/cmd/popplebot/internal/bot"
	"github.com/connorkuehl/popple/cmd/popplebot/internal/command"
	"github.com/connorkuehl/popple/cmd/popplebot/internal/discord"
)

var (
	token = os.Getenv("POPPLEBOT_DISCORD_TOKEN")
	api   = os.Getenv("POPPLEBOT_POPPLE_ENDPOINT")
)

var log = logrus.StandardLogger().WithFields(logrus.Fields{
	"component": "bot",
})

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	client := popple.NewHTTPClient(api)

	log.Info("connecting to Discord")
	session, err := discord.NewDialer(token).Dial()
	if err != nil {
		return err
	}
	defer session.Close()

	router := command.NewRouter("@" + session.State.User.Username)
	disc := discord.NewSession(session)
	bot := bot.New(client, disc, router)

	return bot.Listen(discord.Messages(ctx, session))
}
