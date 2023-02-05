package bot

import (
	"context"
	"errors"
	"strings"
	"text/template"

	"github.com/connorkuehl/popple/internal/command"
	"github.com/connorkuehl/popple/internal/database"
	"github.com/connorkuehl/popple/internal/popple"

	log "github.com/sirupsen/logrus"
)

var (
	templateLevels = template.Must(template.New("levels").Parse(`{{ range $name, $karma := . }}{{ $name }} has {{ $karma }} karma. {{ end }}`))
	templateBoard  = template.Must(template.New("board").Parse(
		`{{ range $entry := . }}* {{ $entry.Who }} has {{ $entry.Karma }} karma.
{{ end }}`))
)

func (b *Bot) handleSetAnnounce(ctx context.Context, args *command.SetAnnounceArgs, guildID, channelID, messageID, content string) {
	ll := log.WithFields(log.Fields{
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
			ll.WithError(err).Error("send message to channel")
		}
		return
	}
	if err != nil {
		ll.WithError(err).Error("unexpected error from arg parser")
	}

	config := popple.ServerConfig{
		ServerID:   guildID,
		NoAnnounce: args.NoAnnounce,
	}

	if err := b.db.PutConfig(ctx, config); err != nil {
		ll.WithError(err).Error("PutConfig")
		return
	}

	if err := b.discord.ReactToMessageWithEmoji(channelID, messageID, "✅"); err != nil {
		ll.WithError(err).Error("react to message in channel")
		return
	}
}

func (b *Bot) handleChangeKarma(ctx context.Context, args *command.ChangeKarmaArgs, guildID, channelID, content string) {
	ll := log.WithFields(log.Fields{
		"guild_id":   guildID,
		"channel_id": channelID,
		"content":    content,
		"handler":    "change_karma",
	})

	_ = args.ParseArg(content)
	if len(args.Increments) == 0 {
		return
	}

	var who []string
	for name := range args.Increments {
		who = append(who, name)
	}

	ents, err := b.db.Entities(ctx, guildID, who...)
	if err != nil {
		ll.WithError(err).Error("Entities")
		return
	}

	levels := make(popple.Increments)
	for name, incr := range args.Increments {
		levels[name] = incr
	}

	for _, ent := range ents {
		levels[ent.Name] += ent.Karma
	}

	ents = nil
	for name, karma := range levels {
		entity := popple.Entity{Name: name, Karma: karma}
		ents = append(ents, entity)
	}

	if err := b.db.PutEntities(ctx, guildID, ents...); err != nil {
		ll.WithError(err).Error("PutEntities")
		return
	}

	config, err := b.db.Config(ctx, guildID)
	if errors.Is(err, database.ErrNotFound) {
		err = nil
	}
	if err != nil {
		ll.WithError(err).Error("Config")
		return
	}

	if config.NoAnnounce {
		return
	}

	for _, ent := range ents {
		levels[ent.Name] = ent.Karma
	}

	var rsp strings.Builder
	err = templateLevels.Execute(&rsp, levels)
	if err != nil {
		ll.WithError(err).Error("apply levels template")
		return
	}

	err = b.discord.SendMessageToChannel(channelID, strings.TrimSpace(rsp.String()))
	if err != nil {
		ll.WithError(err).Error("send message to channel")
		return
	}

}

func (b *Bot) handleCheckKarma(ctx context.Context, args *command.CheckKarmaArgs, guildID, channelID, content string) {
	ll := log.WithFields(log.Fields{
		"guild_id":   guildID,
		"channel_id": channelID,
		"content":    content,
		"handler":    "check_karma",
	})

	_ = args.ParseArg(content)

	if len(args.Who) == 0 {
		return
	}

	ents, err := b.db.Entities(ctx, guildID, args.Who...)
	if err != nil {
		ll.WithError(err).Error("Entities")
		return
	}

	// Since the user input might contain names that don't yet have karma,
	// let's set all inputs to zero and then increment by whatever is in
	// the database.
	levels := make(popple.Increments)
	for _, name := range args.Who {
		levels[name] = 0
	}
	for _, ent := range ents {
		levels[ent.Name] = ent.Karma
	}

	var rsp strings.Builder
	err = templateLevels.Execute(&rsp, levels)
	if err != nil {
		ll.WithError(err).Error("apply levels template")
		return
	}

	err = b.discord.SendMessageToChannel(channelID, strings.TrimSpace(rsp.String()))
	if err != nil {
		ll.WithError(err).Error("send message to channel")
		return
	}
}

func (b *Bot) handleLeaderboard(ctx context.Context, args *command.LeaderboardArgs, guildID, channelID, content string) {
	ll := log.WithFields(log.Fields{
		"guild_id":   guildID,
		"channel_id": channelID,
		"content":    content,
		"handler":    "leaderboard",
	})

	err := args.ParseArg(content)
	if errors.Is(err, command.ErrInvalidArgument) {
		if err := b.discord.SendMessageToChannel(channelID, `Board size must be a positive, non-zero number`); err != nil {
			ll.WithError(err).Error("send message to channel")
		}
		return
	}
	if err != nil {
		ll.WithError(err).Error("unexpected error from arg parser")
		return
	}

	b.handleBoard(ctx, guildID, channelID, content, args.Order, args.Limit)
}

func (b *Bot) handleLoserboard(ctx context.Context, args *command.LoserboardArgs, guildID, channelID, content string) {
	ll := log.WithFields(log.Fields{
		"guild_id":   guildID,
		"channel_id": channelID,
		"content":    content,
		"handler":    "loserboard",
	})

	err := args.ParseArg(content)
	if errors.Is(err, command.ErrInvalidArgument) {
		if err := b.discord.SendMessageToChannel(channelID, `Board size must be a positive, non-zero number`); err != nil {
			ll.WithError(err).Error("send message to channel")
		}
		return
	}
	if err != nil {
		ll.WithError(err).Error("unexpected error from arg parser")
		return
	}

	b.handleBoard(ctx, guildID, channelID, content, args.Order, args.Limit)
}

func (b *Bot) handleBoard(ctx context.Context, guildID, channelID, content string, ord popple.BoardOrder, limit uint) {
	ll := log.WithFields(log.Fields{
		"guild_id":   guildID,
		"channel_id": channelID,
		"content":    content,
		"handler":    "board",
	})

	boardFunc := b.db.Leaderboard
	if ord == popple.BoardOrderAsc {
		boardFunc = b.db.Loserboard
	}

	board, err := boardFunc(ctx, guildID)
	if err != nil {
		ll.WithError(err).Error("board")
		return
	}

	if len(board) == 0 {
		if err := b.discord.SendMessageToChannel(channelID, `No one has any karma yet.`); err != nil {
			ll.WithError(err).Error("failed to send message to Discord channel")
		}
		return
	}

	var r strings.Builder
	err = templateBoard.Execute(&r, board)
	if err != nil {
		ll.WithError(err).Error("failed to apply board template")
		return
	}

	err = b.discord.SendMessageToChannel(channelID, strings.TrimSpace(r.String()))
	if err != nil {
		ll.WithError(err).Error("failed to send message to Discord channel")
		return
	}
}
