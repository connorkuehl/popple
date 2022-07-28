package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/connorkuehl/popple"
	"github.com/connorkuehl/popple/cmd/popplebot/internal/discord"
	poperrs "github.com/connorkuehl/popple/errors"
	"github.com/connorkuehl/popple/event"
)

var (
	templateLevels = template.Must(template.New("levels").Parse(`{{ range $name, $karma := . }}{{ $name }} has {{ $karma }} karma. {{ end }}`))
	templateBoard  = template.Must(template.New("board").Parse(
		`{{ range $entry := . }}* {{ $entry.Name }} has {{ $entry.Karma }} karma.
{{ end }}`))
)

type Service interface {
	HandleCheckedKarma(context.Context, *event.CheckedKarma) error
	HandleCheckedLeaderboard(context.Context, *event.CheckedLeaderboard) error
	HandleCheckedLoserboard(context.Context, *event.CheckedLoserboard) error
	HandleChangedAnnounce(context.Context, *event.ChangedAnnounce) error
	HandleChangedKarma(context.Context, *event.ChangedKarma) error
	HandleDiscordMessage(context.Context, discord.Message) error
}

type EventBus interface {
	PublishRequest(context.Context, *event.Event) error
}

type Discord interface {
	ReactToMessage(channelID, messageID, emoji string) error
	SendMessage(channelID, message string) error
}

type Mux interface {
	Route(body string) (action interface{}, remainder string)
}

type service struct {
	eventbus EventBus
	discord  Discord
	mux      Mux
}

func New(eventbus EventBus, discord Discord, mux Mux) Service {
	return &service{
		eventbus: eventbus,
		discord:  discord,
		mux:      mux,
	}
}

func (s *service) HandleCheckedKarma(ctx context.Context, rsp *event.CheckedKarma) error {
	var r strings.Builder
	err := templateLevels.Execute(&r, rsp.Who)
	if err != nil {
		return err
	}

	err = s.discord.SendMessage(rsp.ReplyTo.ChannelID, r.String())
	return err
}

func (s *service) HandleCheckedLeaderboard(ctx context.Context, rsp *event.CheckedLeaderboard) error {
	var r strings.Builder
	err := templateBoard.Execute(&r, rsp.Board)
	if err != nil {
		return err
	}

	err = s.discord.SendMessage(rsp.ReplyTo.ChannelID, r.String())
	return err
}

func (s *service) HandleCheckedLoserboard(ctx context.Context, rsp *event.CheckedLoserboard) error {
	var r strings.Builder
	err := templateBoard.Execute(&r, rsp.Board)
	if err != nil {
		return err
	}

	err = s.discord.SendMessage(rsp.ReplyTo.ChannelID, r.String())
	return err
}

func (s *service) HandleChangedAnnounce(ctx context.Context, rsp *event.ChangedAnnounce) error {
	return s.discord.ReactToMessage(rsp.ReactTo.ChannelID, rsp.ReactTo.MessageID, "✅")
}

func (s *service) HandleChangedKarma(ctx context.Context, rsp *event.ChangedKarma) error {
	if !rsp.Announce {
		return nil
	}

	var r strings.Builder
	err := templateLevels.Execute(&r, rsp.Who)
	if err != nil {
		return err
	}

	return s.discord.SendMessage(rsp.ReplyTo.ChannelID, r.String())
}

func (s *service) HandleDiscordMessage(ctx context.Context, msg discord.Message) error {
	var req *event.Event

	action, body := s.mux.Route(msg.Body)

	switch action.(type) {
	case popple.AnnounceHandler:
		on, err := popple.ParseAnnounceArgs(body)
		if errors.Is(err, poperrs.ErrMissingArgument) || errors.Is(err, poperrs.ErrInvalidArgument) {
			err := s.discord.ReactToMessage(msg.ChannelID, msg.ID, "❓")
			if err != nil {
				return fmt.Errorf("failed to react to message: %w", err)
			}
			err = s.discord.SendMessage(msg.ChannelID, `Valid announce settings are: "on", "off", "yes", "no"`)
			if err != nil {
				err = fmt.Errorf("failed to send message: %w", err)
			}
			return err
		}

		req = &event.Event{
			RequestChangeAnnounce: &event.RequestChangeAnnounce{
				ReactTo: event.ReactTo{
					ChannelID: msg.ChannelID,
					MessageID: msg.ID,
				},
				ServerID:   msg.ServerID,
				NoAnnounce: !on,
			}}
	case popple.BumpKarmaHandler:
		increments, _ := popple.ParseBumpKarmaArgs(body)
		for k, v := range increments {
			if v == 0 {
				delete(increments, k)
			}
		}
		if len(increments) == 0 {
			return nil
		}

		req = &event.Event{
			RequestBumpKarma: &event.RequestBumpKarma{
				ReplyTo: event.ReplyTo{
					ChannelID: msg.ChannelID,
				},
				ServerID: msg.ServerID,
				Who:      increments,
			}}
	case popple.KarmaHandler:
		who, err := popple.ParseKarmaArgs(body)
		if err != nil {
			err = s.discord.ReactToMessage(msg.ChannelID, msg.ID, "❓")
			if err != nil {
				err = fmt.Errorf("message reaction add failed: %w", err)
			}
			return err
		}

		req = &event.Event{
			RequestCheckKarma: &event.RequestCheckKarma{
				ReplyTo: event.ReplyTo{
					ChannelID: msg.ChannelID,
				},
				ServerID: msg.ServerID,
				Who:      who,
			}}
	case popple.LeaderboardHandler:
		limit, err := popple.ParseLeaderboardArgs(body)
		if errors.Is(err, poperrs.ErrInvalidArgument) {
			err := s.discord.SendMessage(msg.ChannelID, "The number of entries to list must be a positive non-zero integer")
			if err != nil {
				err = fmt.Errorf("message send failed: %w", err)
			}
			return err
		}

		req = &event.Event{
			RequestCheckLeaderboard: &event.RequestCheckLeaderboard{
				ReplyTo: event.ReplyTo{
					ChannelID: msg.ChannelID,
				},
				ServerID: msg.ServerID,
				Limit:    limit,
			}}
	case popple.LoserboardHandler:
		limit, err := popple.ParseLoserboardArgs(body)
		if errors.Is(err, poperrs.ErrInvalidArgument) {
			err := s.discord.SendMessage(msg.ChannelID, "The number of entries to list must be a positive non-zero integer")
			if err != nil {
				err = fmt.Errorf("message send failed: %w", err)
			}
			return err
		}

		req = &event.Event{
			RequestCheckLoserboard: &event.RequestCheckLoserboard{
				ReplyTo: event.ReplyTo{
					ChannelID: msg.ChannelID,
				},
				ServerID: msg.ServerID,
				Limit:    limit,
			}}
	default:
		return fmt.Errorf("bug: reached default case for handle message: %q", body)
	}

	return s.eventbus.PublishRequest(ctx, req)

}
