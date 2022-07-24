package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "modernc.org/sqlite"

	"github.com/connorkuehl/popple"
	"github.com/connorkuehl/popple/cmd/popple/internal/service"
	sqliterepo "github.com/connorkuehl/popple/repo/sqlite"
)

var (
	sqlitedb     = os.Getenv("POPPLE_SQLITE_DB")
	token        = os.Getenv("POPPLE_DISCORD_BOT_TOKEN")
	listenHealth = os.Getenv("POPPLE_LISTEN_HEALTH")
)

type responseWriter struct {
	m *discordgo.Message
	s *discordgo.Session
}

func (r responseWriter) React(emoji string) error {
	err := r.s.MessageReactionAdd(r.m.ChannelID, r.m.ID, emoji)
	return err
}

func (r responseWriter) SendMessage(msg string) error {
	_, err := r.s.ChannelMessageSend(r.m.ChannelID, msg)
	return err
}

type discord struct {
	s *discordgo.Session
}

func (d discord) HeartbeatLatency() time.Duration {
	return d.s.HeartbeatLatency()
}

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	db, err := sql.Open("sqlite", sqlitedb)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return err
	}

	err = session.Open()
	if err != nil {
		return err
	}
	defer session.Close()
	log.Println("connected to discord")

	exiting := make(chan struct{})

	repo, err := sqliterepo.NewRepository(db)
	if err != nil {
		return fmt.Errorf("failed to init repo: %w", err)
	}

	disc := discord{session}

	var svc service.Service = service.New(repo, disc)
	svc = service.NewLogged(svc)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		details, ok := svc.Health()
		details["healthy"] = ok

		if err := json.NewEncoder(w).Encode(details); err != nil {
			log.Printf("failed to encode health checks: %v", err)
		}
	})
	go func() {
		log.Println("health checks:", listenHealth+"/health")
		log.Println(http.ListenAndServe(listenHealth, nil))
	}()

	mux := popple.NewMux("@" + session.State.User.Username)
	detachMessageCreateHandler := session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		select {
		case <-exiting:
			return
		default:
		}

		if s.State.User.ID == m.Author.ID {
			return
		}

		isDM := len(m.GuildID) == 0
		if isDM {
			return
		}

		message := strings.TrimSpace(m.ContentWithMentionsReplaced())
		action, body := mux.Route(message)
		request := service.Request{
			ServerID: m.GuildID,
			Message:  body,
		}
		response := responseWriter{
			m: m.Message,
			s: s,
		}

		switch action.(type) {
		case popple.AnnounceHandler:
			_ = svc.Announce(request, response)
		case popple.BumpKarmaHandler:
			_ = svc.BumpKarma(request, response)
		case popple.KarmaHandler:
			_ = svc.Karma(request, response)
		case popple.LeaderboardHandler:
			_ = svc.Leaderboard(request, response)
		case popple.LoserboardHandler:
			_ = svc.Loserboard(request, response)
		}
	})
	defer detachMessageCreateHandler()
	log.Println("ready to dole out some karma")

	plsStop := make(chan os.Signal, 1)
	signal.Notify(plsStop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-plsStop

	log.Println("shutting down due to signal")

	close(exiting)

	return nil
}
