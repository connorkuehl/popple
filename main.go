package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT)
	defer cancel()

	if err := run(ctx); err != nil {
		log.WithError(err).Error("shutting down")
	}
}

func run(ctx context.Context) error {
	bot, cleanup, err := InitializeBot()
	if err != nil {
		return err
	}
	defer cleanup()

	return bot.Listen(ctx)
}
