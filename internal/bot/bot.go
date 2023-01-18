package bot

import (
	"context"
	"errors"

	"github.com/connorkuehl/popple/internal/command"
	"github.com/connorkuehl/popple/internal/discord"
	"github.com/connorkuehl/popple/internal/popple"
)

type Session interface {
	SendMessageToChannel(channelID string, msg string) error
	ReactToMessageWithEmoji(channelID, messageID, emojiID string) error
	Messages() <-chan discord.Message
}

type DB interface {
	Config(ctx context.Context, serverID string) (popple.ServerConfig, error)
	PutConfig(context.Context, popple.ServerConfig) error
	Entities(ctx context.Context, serverID string, names ...string) ([]popple.Entity, error)
	PutEntities(ctx context.Context, serverID string, entities ...popple.Entity) error
	Leaderboard(ctx context.Context, serverID string) (popple.Board, error)
	Loserboard(ctx context.Context, serverID string) (popple.Board, error)
}

type CommandRouter interface {
	Route(s string) (args command.ArgParser, remainder string)
}

type Bot struct {
	discord Session
	db      DB
	router  CommandRouter
}

func New(discord Session, db DB, router CommandRouter) *Bot {
	return &Bot{
		discord: discord,
		db:      db,
		router:  router,
	}
}

func (b *Bot) Listen(ctx context.Context) error {
	messages := b.discord.Messages()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-messages:
			if !ok {
				return errors.New("discord message stream closed")
			}

			cmd, remainder := b.router.Route(msg.Content)

			switch c := cmd.(type) {
			case *command.SetAnnounceArgs:
				b.handleSetAnnounce(ctx, c, msg.GuildID, msg.ChannelID, msg.ID, remainder)

			case *command.ChangeKarmaArgs:
				b.handleChangeKarma(ctx, c, msg.GuildID, msg.ChannelID, remainder)

			case *command.CheckKarmaArgs:
				b.handleCheckKarma(ctx, c, msg.GuildID, msg.ChannelID, remainder)

			case *command.LeaderboardArgs:
				b.handleLeaderboard(ctx, c, msg.GuildID, msg.ChannelID, remainder)

			case *command.LoserboardArgs:
				b.handleLoserboard(ctx, c, msg.GuildID, msg.ChannelID, remainder)
			}
		}
	}
}
