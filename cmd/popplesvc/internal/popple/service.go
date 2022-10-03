package popple

import (
	"context"

	"github.com/connorkuehl/popple"
)

type Store interface {
	Board(ctx context.Context, serverID string, ord popple.BoardOrder, limit uint) (popple.Board, error)
	ChangeKarma(ctx context.Context, serverID string, increments popple.Increments) (popple.Increments, error)
	CheckKarma(ctx context.Context, serverID string, who []string) (map[string]int64, error)
	Config(ctx context.Context, serverID string) (*popple.Config, error)
	PutConfig(ctx context.Context, config *popple.Config) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{
		store: store,
	}
}

func (s *Service) Board(ctx context.Context, serverID string, ord popple.BoardOrder, limit uint) (popple.Board, error) {
	return s.store.Board(ctx, serverID, ord, limit)
}

func (s *Service) ChangeKarma(ctx context.Context, serverID string, increments popple.Increments) (popple.Increments, error) {
	filtered := make(popple.Increments)
	for name, karma := range increments {
		if karma == 0 {
			continue
		}

		filtered[name] = karma
	}
	return s.store.ChangeKarma(ctx, serverID, filtered)
}

func (s *Service) CheckKarma(ctx context.Context, serverID string, who []string) (map[string]int64, error) {
	return s.store.CheckKarma(ctx, serverID, who)
}

func (s *Service) Config(ctx context.Context, serverID string) (*popple.Config, error) {
	return s.store.Config(ctx, serverID)
}

func (s *Service) PutConfig(ctx context.Context, config *popple.Config) error {
	return s.store.PutConfig(ctx, config)
}
