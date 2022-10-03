package popplehttp

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/connorkuehl/popple"
)

var log = logrus.StandardLogger().WithFields(logrus.Fields{
	"component": "popplehttp",
})

type Service interface {
	Board(ctx context.Context, serverID string, ord popple.BoardOrder, limit uint) (popple.Board, error)
	ChangeKarma(ctx context.Context, serverID string, increments popple.Increments) (popple.Increments, error)
	CheckKarma(ctx context.Context, serverID string, who []string) (map[string]int64, error)
	Config(ctx context.Context, serverID string) (*popple.Config, error)
	PutConfig(ctx context.Context, config *popple.Config) error
}

type Server struct {
	svc Service
}

func NewServer(svc Service) *Server {
	return &Server{
		svc: svc,
	}
}

func (s *Server) Board(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(logrus.Fields{
		"handler": "board",
	})

	serverID := chi.URLParam(r, "server_id")

	var opts struct {
		Ord   string `json:"order"`
		Limit uint   `json:"limit"`
	}

	err := json.NewDecoder(r.Body).Decode(&opts)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	boardOrd := popple.BoardOrderAsc
	switch opts.Ord {
	case "asc":
	case "desc":
		boardOrd = popple.BoardOrderDsc
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if opts.Limit < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	board, err := s.svc.Board(ctx, serverID, boardOrd, opts.Limit)
	if err != nil {
		logger.WithError(err).Error("Board failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(board)
	if err != nil {
		logger.WithError(err).Error("json encoding failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *Server) ChangeKarma(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(logrus.Fields{
		"handler": "change_karma",
	})

	serverID := chi.URLParam(r, "server_id")

	var increments popple.Increments
	err := json.NewDecoder(r.Body).Decode(&increments)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	newLevels, err := s.svc.ChangeKarma(ctx, serverID, increments)
	if err != nil {
		logger.WithError(err).Error("ChangeKarma failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(newLevels)
	if err != nil {
		logger.WithError(err).Error("json encoding failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *Server) CheckKarma(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(logrus.Fields{
		"handler": "check_karma",
	})

	serverID := chi.URLParam(r, "server_id")

	var who []string
	err := json.NewDecoder(r.Body).Decode(&who)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	levels, err := s.svc.CheckKarma(ctx, serverID, who)
	if err != nil {
		logger.WithError(err).Error("CheckKarma failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(levels)
	if err != nil {
		logger.WithError(err).Error("json encoding failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *Server) Config(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(logrus.Fields{
		"handler": "config",
	})

	serverID := chi.URLParam(r, "server_id")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	config, err := s.svc.Config(ctx, serverID)
	if err != nil {
		logger.WithError(err).Error("Config failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(config)
	if err != nil {
		logger.WithError(err).Error("json encoding failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *Server) PutConfig(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(logrus.Fields{
		"handler": "put_config",
	})

	serverID := chi.URLParam(r, "server_id")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var config popple.Config
	err := json.NewDecoder(r.Body).Decode(&config)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	config.ServerID = serverID

	err = s.svc.PutConfig(ctx, &config)
	if err != nil {
		logger.WithError(err).Error("PutConfig failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
