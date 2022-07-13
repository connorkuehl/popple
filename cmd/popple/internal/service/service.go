package service

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/connorkuehl/popple"
	poperr "github.com/connorkuehl/popple/errors"
	poppleget "github.com/connorkuehl/popple/get"
	"text/template"
)

var (
	levelsTemplate = template.Must(template.New("levels").Parse(`{{ range $name, $karma := . }}{{ $name }} has {{ $karma }} karma. {{ end }}`))
	boardTemplate  = template.Must(template.New("board").Parse(
		`{{ range $entry := . }}* {{ $entry.Name }} has {{ $entry.Karma }} karma.
{{ end }}`))
)

type Request struct {
	ServerID string
	Message  string
}

type ResponseWriter interface {
	React(emoji string) error
	SendMessage(msg string) error
}

type Service interface {
	Announce(req Request, rsp ResponseWriter) error
	BumpKarma(req Request, rsp ResponseWriter) error
	Karma(req Request, rsp ResponseWriter) error
	Leaderboard(req Request, rsp ResponseWriter) error
	Loserboard(req Request, rsp ResponseWriter) error
}

type service struct {
	mu sync.Mutex

	repo popple.Repository
}

func New(repo popple.Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) Announce(req Request, rsp ResponseWriter) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	on, err := popple.ParseAnnounceArgs(req.Message)
	if errors.Is(err, poperr.ErrMissingArgument) || errors.Is(err, poperr.ErrInvalidArgument) {
		err = rsp.SendMessage(`Valid announce settings are: "on", "off", "yes", "no"`)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
		return nil
	}

	err = popple.Announce(s.repo, req.Message, on)
	if err != nil {
		return fmt.Errorf("failed to set announce: %w", err)
	}

	err = rsp.React("✅")
	if err != nil {
		return fmt.Errorf("failed to add reaction to message: %w", err)
	}

	return nil
}

func (s *service) BumpKarma(req Request, rsp ResponseWriter) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	increments, _ := popple.ParseBumpKarmaArgs(req.Message)

	levels, err := popple.BumpKarma(s.repo, req.ServerID, increments)
	if err != nil {
		return fmt.Errorf("failed to bump karma: %w", err)
	}

	config, err := s.repo.Config(req.ServerID)
	if errors.Is(err, poperr.ErrNotFound) {
		err = nil
	}
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if config.NoAnnounce {
		return nil
	}

	if len(levels) < 1 {
		return nil
	}

	var r strings.Builder
	err = levelsTemplate.Execute(&r, levels)
	if err != nil {
		return fmt.Errorf("failed to apply template response: %w", err)
	}

	err = rsp.SendMessage(r.String())
	if err != nil {
		return fmt.Errorf("failed to send message to channel: %w", err)
	}

	return nil
}

func (s *service) Karma(req Request, rsp ResponseWriter) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	who, err := popple.ParseKarmaArgs(req.Message)
	if err != nil {
		err = rsp.React("❓")
		if err != nil {
			return fmt.Errorf("failed to add reaction to message: %w", err)
		}
		return nil
	}

	levels, err := popple.Karma(s.repo, req.ServerID, who)
	if err != nil {
		return fmt.Errorf("failed to check karma: %w", err)
	}

	if len(levels) < 1 {
		return nil
	}

	var r strings.Builder
	err = levelsTemplate.Execute(&r, levels)
	if err != nil {
		return fmt.Errorf("failed to apply template response: %w", err)
	}

	err = rsp.SendMessage(r.String())
	if err != nil {
		return fmt.Errorf("failed to send message to channel: %w", err)
	}

	return nil
}

type boardOrdering func(repo poppleget.EntityRepository, serverID string, limit uint) (board []poppleget.Entity, err error)

func (s *service) Leaderboard(req Request, rsp ResponseWriter) error {
	return s.board(req, rsp, popple.Leaderboard)
}

func (s *service) Loserboard(req Request, rsp ResponseWriter) error {
	return s.board(req, rsp, popple.Loserboard)
}

func (s *service) board(req Request, rsp ResponseWriter, ordering boardOrdering) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	limit, err := popple.ParseLeaderboardArgs(req.Message)
	if errors.Is(err, poperr.ErrInvalidArgument) {
		err = rsp.SendMessage("The number of entries to list must be a positive non-zero integer")
		if err != nil {
			return fmt.Errorf("failed to send message to channel: %w", err)
		}
	}

	board, err := ordering(s.repo, req.ServerID, limit)
	if err != nil {
		return err
	}

	if len(board) < 1 {
		return nil
	}

	var r strings.Builder
	err = boardTemplate.Execute(&r, board)
	if err != nil {
		return fmt.Errorf("failed to apply template response: %w", err)
	}

	err = rsp.SendMessage(r.String())
	if err != nil {
		return fmt.Errorf("failed to send message to channel: %w", err)
	}

	return nil
}
