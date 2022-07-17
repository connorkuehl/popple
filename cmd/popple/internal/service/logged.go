package service

import (
	"encoding/json"
	"log"
	"strings"
)

type loggedService struct {
	svc Service
}

func NewLogged(svc Service) *loggedService {
	return &loggedService{svc: svc}
}

func (s *loggedService) Health() (details map[string]interface{}, ok bool) {
	details, ok = s.svc.Health()

	report := struct {
		OK      bool                   `json:"ok"`
		Details map[string]interface{} `json:"details"`
	}{
		OK:      ok,
		Details: details,
	}

	if !ok {
		var sb strings.Builder
		_ = json.NewEncoder(&sb).Encode(report)
		log.Println("health check failed", sb.String())
	}

	return details, ok
}

func (s *loggedService) Announce(req Request, rsp ResponseWriter) error {
	err := s.svc.Announce(req, rsp)
	if err != nil {
		log.Println("announce", err)
	}
	return err
}

func (s *loggedService) BumpKarma(req Request, rsp ResponseWriter) error {
	err := s.svc.BumpKarma(req, rsp)
	if err != nil {
		log.Println("bump_karma", err)
	}
	return err
}

func (s *loggedService) Karma(req Request, rsp ResponseWriter) error {
	err := s.svc.Karma(req, rsp)
	if err != nil {
		log.Println("karma", err)
	}
	return err
}

func (s *loggedService) Leaderboard(req Request, rsp ResponseWriter) error {
	err := s.svc.Leaderboard(req, rsp)
	if err != nil {
		log.Println("leaderboard", err)
	}
	return err
}

func (s *loggedService) Loserboard(req Request, rsp ResponseWriter) error {
	err := s.svc.Loserboard(req, rsp)
	if err != nil {
		log.Println("loserboard", err)
	}
	return err
}
