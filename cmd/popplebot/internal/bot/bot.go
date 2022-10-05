package bot

import (
	"context"
	"errors"
	"strings"
	"text/template"

	"github.com/sirupsen/logrus"

	"github.com/connorkuehl/popple"
	"github.com/connorkuehl/popple/cmd/popplebot/internal/command"
	"github.com/connorkuehl/popple/cmd/popplebot/internal/discord"
)

var (
	templateLevels = template.Must(template.New("levels").Parse(`{{ range $name, $karma := . }}{{ $name }} has {{ $karma }} karma. {{ end }}`))
	templateBoard  = template.Must(template.New("board").Parse(
		`{{ range $entry := . }}* {{ $entry.Who }} has {{ $entry.Karma }} karma.
{{ end }}`))
)

var log = logrus.StandardLogger().WithFields(logrus.Fields{
	"component": "bot",
})

//go:generate mockery --name PoppleClient --case underscore --with-expecter --testonly --inpackage
type PoppleClient interface {
	Board(ctx context.Context, serverID string, ord popple.BoardOrder, limit uint) (popple.Board, error)
	ChangeKarma(ctx context.Context, serverID string, increments popple.Increments) (popple.Increments, error)
	CheckKarma(ctx context.Context, serverID string, who []string) (map[string]int64, error)
	Config(ctx context.Context, serverID string) (*popple.Config, error)
	PutConfig(ctx context.Context, config *popple.Config) error
}

//go:generate mockery --name Discord --case underscore --with-expecter --testonly --inpackage
type Discord interface {
	SendMessageToChannel(channelID string, msg string) error
	ReactToMessageWithEmoji(channelID, messageID, emojiID string) error
}

type Bot struct {
	client  PoppleClient
	discord Discord
	router  *command.Router
}

func New(client PoppleClient, discord Discord, router *command.Router) *Bot {
	return &Bot{
		client:  client,
		discord: discord,
		router:  router,
	}
}

func (b *Bot) Listen(messages <-chan discord.Message) error {
	log.Info("ready to process Discord messages")

	for msg := range messages {
		cmd, remainder := b.router.Route(msg.Content)

		switch c := cmd.(type) {
		case *command.SetAnnounceArgs:
			b.handleSetAnnounce(c, msg.GuildID, msg.ChannelID, msg.ID, remainder)

		case *command.ChangeKarmaArgs:
			b.handleChangeKarma(c, msg.GuildID, msg.ChannelID, remainder)

		case *command.CheckKarmaArgs:
			b.handleCheckKarma(c, msg.GuildID, msg.ChannelID, remainder)

		case *command.LeaderboardArgs:
			b.handleLeaderboard(c, msg.GuildID, msg.ChannelID, remainder)

		case *command.LoserboardArgs:
			b.handleLoserboard(c, msg.GuildID, msg.ChannelID, remainder)
		}
	}

	return nil
}

func (b *Bot) handleSetAnnounce(args *command.SetAnnounceArgs, guildID, channelID, messageID, content string) {
	logger := log.WithFields(logrus.Fields{
		"guild_id":   guildID,
		"channel_id": channelID,
		"message_id": messageID,
		"content":    content,
		"handler":    "set_announce",
	})

	err := args.ParseArg(content)
	switch {
	case errors.Is(err, command.ErrInvalidArgument), errors.Is(err, command.ErrMissingArgument):
		if err := b.discord.SendMessageToChannel(channelID, `Valid announce settings are "yes", "on", "no", "off"`); err != nil {
			logger.WithError(err).Error("failed to send message to Discord channel")
		}
		return
	}
	if err != nil {
		logger.WithError(err).Error("unexpected error from arg parser")
	}

	config := popple.Config{
		ServerID:   guildID,
		NoAnnounce: args.NoAnnounce,
	}
	err = b.client.PutConfig(context.Background(), &config)
	if err != nil {
		logger.WithError(err).Error("client PutConfig failed")
		return
	}

	err = b.discord.ReactToMessageWithEmoji(channelID, messageID, "✅")
	if err != nil {
		logger.WithError(err).Error("failed to send message to Discord channel")
		return
	}
}

func (b *Bot) handleChangeKarma(args *command.ChangeKarmaArgs, guildID, channelID, content string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := log.WithFields(logrus.Fields{
		"guild_id":   guildID,
		"channel_id": channelID,
		"content":    content,
		"handler":    "change_karma",
	})

	_ = args.ParseArg(content)

	levels, err := b.client.ChangeKarma(ctx, guildID, args.Increments)
	if err != nil {
		logger.WithError(err).Error("client ChangeKarma failed")
		return
	}

	config, err := b.client.Config(ctx, guildID)
	if err != nil {
		logger.WithError(err).Error("client Config failed")
		return
	}

	if config.NoAnnounce {
		return
	}

	var rsp strings.Builder
	err = templateLevels.Execute(&rsp, levels)
	if err != nil {
		logger.WithError(err).Error("failed to apply levels template")
		return
	}

	err = b.discord.SendMessageToChannel(channelID, rsp.String())
	if err != nil {
		logger.WithError(err).Error("failed to send message to Discord channel")
		return
	}
}

func (b *Bot) handleCheckKarma(args *command.CheckKarmaArgs, guildID, channelID, content string) {
	logger := log.WithFields(logrus.Fields{
		"guild_id":   guildID,
		"channel_id": channelID,
		"content":    content,
		"handler":    "check_karma",
	})

	_ = args.ParseArg(content)

	levels, err := b.client.CheckKarma(context.Background(), guildID, args.Who)
	if err != nil {
		logger.WithError(err).Error("client CheckKarma failed")
		return
	}

	var rsp strings.Builder
	err = templateLevels.Execute(&rsp, levels)
	if err != nil {
		logger.WithError(err).Error("failed to apply levels template")
		return
	}

	err = b.discord.SendMessageToChannel(channelID, rsp.String())
	if err != nil {
		logger.WithError(err).Error("failed to apply levels template")
		return
	}
}

func (b *Bot) handleLeaderboard(args *command.LeaderboardArgs, guildID, channelID, content string) {
	logger := log.WithFields(logrus.Fields{
		"guild_id":   guildID,
		"channel_id": channelID,
		"content":    content,
		"handler":    "leaderboard",
	})

	err := args.ParseArg(content)
	if errors.Is(err, command.ErrInvalidArgument) {
		if err := b.discord.SendMessageToChannel(channelID, `Board size must be a positive, non-zero number`); err != nil {
			logger.WithError(err).Error("failed to send message to Discord channel")
		}
		return
	}
	if err != nil {
		logger.WithError(err).Error("unexpected error from arg parser")
		return
	}

	b.handleBoard(guildID, channelID, content, args.Order, args.Limit)
}

func (b *Bot) handleLoserboard(args *command.LoserboardArgs, guildID, channelID, content string) {
	logger := log.WithFields(logrus.Fields{
		"guild_id":   guildID,
		"channel_id": channelID,
		"content":    content,
		"handler":    "loserboard",
	})

	err := args.ParseArg(content)
	if errors.Is(err, command.ErrInvalidArgument) {
		if err := b.discord.SendMessageToChannel(channelID, `Board size must be a positive, non-zero number`); err != nil {
			logger.WithError(err).Error("failed to send message to Discord channel")
		}
		return
	}
	if err != nil {
		logger.WithError(err).Error("unexpected error from arg parser")
		return
	}

	b.handleBoard(guildID, channelID, content, args.Order, args.Limit)
}

func (b *Bot) handleBoard(guildID, channelID, content string, ord popple.BoardOrder, limit uint) {
	logger := log.WithFields(logrus.Fields{
		"guild_id":   guildID,
		"channel_id": channelID,
		"content":    content,
		"handler":    "board",
	})

	board, err := b.client.Board(context.Background(), guildID, ord, limit)
	if err != nil {
		logger.WithError(err).Error("client Board failed")
		return
	}

	if len(board) == 0 {
		if err := b.discord.SendMessageToChannel(channelID, `No one has any karma yet.`); err != nil {
			logger.WithError(err).Error("failed to send message to Discord channel")
		}
		return
	}

	var r strings.Builder
	err = templateBoard.Execute(&r, board)
	if err != nil {
		logger.WithError(err).Error("failed to apply board template")
		return
	}

	err = b.discord.SendMessageToChannel(channelID, r.String())
	if err != nil {
		logger.WithError(err).Error("failed to send message to Discord channel")
		return
	}
}
