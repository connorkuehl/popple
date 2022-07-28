package service

import (
	"context"
	"errors"

	"github.com/connorkuehl/popple"
	"github.com/connorkuehl/popple/internal/event"
)

type Service interface {
	CheckKarma(context.Context, *event.RequestCheckKarma) error
	CheckLeaderboard(context.Context, *event.RequestCheckLeaderboard) error
	CheckLoserboard(context.Context, *event.RequestCheckLoserboard) error
	ChangeAnnounce(context.Context, *event.RequestChangeAnnounce) error
	ChangeKarma(context.Context, *event.RequestBumpKarma) error
}

type EventBus interface {
	EmitEvent(context.Context, string, *event.Event) error
}

type Repo popple.Repository

type service struct {
	eventbus EventBus
	repo     Repo
}

func New(eventbus EventBus, repo Repo) Service {
	return &service{
		eventbus: eventbus,
		repo:     repo,
	}
}

func (s *service) CheckKarma(ctx context.Context, req *event.RequestCheckKarma) error {
	who, err := popple.Karma(s.repo, req.ServerID, req.Who)
	if err != nil {
		return err
	}

	return s.eventbus.EmitEvent(ctx, "checked.karma", &event.Event{
		CheckedKarma: &event.CheckedKarma{
			ReplyTo: req.ReplyTo,
			Who:     who,
		},
	})
}

func (s *service) CheckLeaderboard(ctx context.Context, req *event.RequestCheckLeaderboard) error {
	top, err := popple.Leaderboard(s.repo, req.ServerID, req.Limit)
	if err != nil {
		return err
	}

	board := make([]event.Score, 0, len(top))
	for _, s := range top {
		board = append(board, event.Score{Name: s.Name, Karma: s.Karma})
	}

	return s.eventbus.EmitEvent(ctx, "checked.leaderboard", &event.Event{
		CheckedLeaderboard: &event.CheckedLeaderboard{
			ReplyTo: req.ReplyTo,
			Board:   board,
		},
	})
}

func (s *service) CheckLoserboard(ctx context.Context, req *event.RequestCheckLoserboard) error {
	top, err := popple.Loserboard(s.repo, req.ServerID, req.Limit)
	if err != nil {
		return err
	}

	board := make([]event.Score, 0, len(top))
	for _, s := range top {
		board = append(board, event.Score{Name: s.Name, Karma: s.Karma})
	}

	return s.eventbus.EmitEvent(ctx, "checked.loserboard", &event.Event{
		CheckedLoserboard: &event.CheckedLoserboard{
			ReplyTo: req.ReplyTo,
			Board:   board,
		},
	})
}

func (s *service) ChangeAnnounce(ctx context.Context, req *event.RequestChangeAnnounce) error {
	err := popple.Announce(s.repo, req.ServerID, !req.NoAnnounce)
	if err != nil {
		return err
	}

	return s.eventbus.EmitEvent(ctx, "changed.announce", &event.Event{
		ChangedAnnounce: &event.ChangedAnnounce{
			ReactTo: req.ReactTo,
		},
	})
}

func (s *service) ChangeKarma(ctx context.Context, req *event.RequestBumpKarma) error {
	incr, err := popple.BumpKarma(s.repo, req.ServerID, req.Who)
	if err != nil {
		return err
	}

	if len(incr) == 0 {
		return nil
	}

	cfg, err := s.repo.Config(req.ServerID)
	if errors.Is(err, popple.ErrNotFound) {
		err = nil
	}
	if err != nil {
		return err
	}

	return s.eventbus.EmitEvent(ctx, "changed.karma", &event.Event{
		ChangedKarma: &event.ChangedKarma{
			ReplyTo:  req.ReplyTo,
			Who:      incr,
			Announce: !cfg.NoAnnounce,
		},
	})
}
