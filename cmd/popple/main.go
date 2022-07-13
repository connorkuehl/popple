package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/alecthomas/kingpin.v2"
	_ "modernc.org/sqlite"

	"github.com/connorkuehl/popple"
	"github.com/connorkuehl/popple/cmd/popple/internal/service"
	"github.com/connorkuehl/popple/config"
	sqliterepo "github.com/connorkuehl/popple/repo/sqlite"
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

func main() {
	if err := configureAndRun(); err != nil {
		log.Fatalln(err)
	}
}

func configureAndRun() error {
	var (
		cfg        = config.Config{}
		configFile = kingpin.Flag("config", "Path to the Popple config.").ExistingFile()
	)

	kingpin.Flag("database", "Path to the SQLite database.").StringVar(&cfg.DBPath)
	kingpin.Flag("token", "Path to the Discord bot token file.").StringVar(&cfg.Token)
	kingpin.Parse()

	// Popple config path was specified in the command line arguments.
	if *configFile != "" {
		loaded, err := config.LoadFromFile(*configFile)
		if err != nil {
			return fmt.Errorf("failed to read popple config: %w", err)
		}

		// Merge the loaded config with the config that was already
		// set via command line arguments, but the command line-supplied
		// arguments taken precedent.
		if cfg.Token == "" {
			cfg.Token = loaded.Token
		}
		if cfg.DBPath == "" {
			cfg.DBPath = loaded.DBPath
		}
	}

	// Make sure we have all the required config.
	switch {
	case cfg.Token == "":
		return errors.New("token is missing from config")
	case cfg.DBPath == "":
		return errors.New("database path missing from config")
	}

	return run(cfg)
}

func run(cfg config.Config) error {
	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	session, err := discordgo.New("Bot " + cfg.Token)
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

	var svc service.Service = service.New(repo)
	svc = service.NewLogged(svc)

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
