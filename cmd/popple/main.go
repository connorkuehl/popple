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
	"text/template"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/alecthomas/kingpin.v2"
	_ "modernc.org/sqlite"

	"github.com/connorkuehl/popple"
	"github.com/connorkuehl/popple/config"
	poperr "github.com/connorkuehl/popple/errors"
	sqliterepo "github.com/connorkuehl/popple/repo/sqlite"
)

var (
	levelsTemplate = template.Must(template.New("levels").Parse(`{{ range $name, $karma := . }}{{ $name }} has {{ $karma }} karma. {{ end }}`))
	boardTemplate  = template.Must(template.New("board").Parse(
		`{{ range $entry := . }}* {{ $entry.Name }} has {{ $entry.Karma }} karma.
{{ end }}`))
)

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

	repoMu := make(chan struct{}, 1)
	repo, err := sqliterepo.NewRepository(db)
	if err != nil {
		return fmt.Errorf("failed to init repo: %w", err)
	}

	mux := popple.NewMux("@" + session.State.User.Username)

	detachMessageCreateHandler := session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		select {
		case <-exiting:
			return
		default:
		}

		repoMu <- struct{}{}
		defer func() { <-repoMu }()

		if s.State.User.ID == m.Author.ID {
			return
		}

		isDM := len(m.GuildID) == 0
		if isDM {
			return
		}

		message := strings.TrimSpace(m.ContentWithMentionsReplaced())
		action, body := mux.Route(message)

		switch action := action.(type) {
		case popple.AnnounceHandler:
			on, err := popple.ParseAnnounceArgs(body)
			if errors.Is(err, poperr.ErrMissingArgument) || errors.Is(err, poperr.ErrInvalidArgument) {
				_, err = s.ChannelMessageSend(m.ChannelID, `Valid announce settings are: "on", "off", "yes", "no"`)
				if err != nil {
					log.Println("failed to send message", err)
				}
				return
			}
			if err != nil {
				log.Println("ParseAnnounceArgs", err)
				return
			}

			err = action(repo, m.GuildID, on)
			if err != nil {
				log.Println("AnnounceHandler", err)
				return
			}

			err = s.MessageReactionAdd(m.ChannelID, m.ID, "✅")
			if err != nil {
				log.Println("failed to add reaction to message", err)
				return
			}
		case popple.BumpKarmaHandler:
			increments, _ := popple.ParseBumpKarmaArgs(body)

			levels, err := action(repo, m.GuildID, increments)
			if err != nil {
				log.Println("BumpKarmaHandler", err)
				return
			}

			config, err := repo.Config(m.GuildID)
			if errors.Is(err, poperr.ErrNotFound) {
				err = nil
			}

			if err != nil {
				log.Println("repo.Config", err)
				return
			}

			if config.NoAnnounce {
				return
			}

			if len(levels) < 1 {
				return
			}

			var r strings.Builder
			err = levelsTemplate.Execute(&r, levels)
			if err != nil {
				log.Println("levelsTemplate.Execute", err)
				return
			}

			_, err = s.ChannelMessageSend(m.ChannelID, r.String())
			if err != nil {
				log.Println("failed to send message to channel", err)
				return
			}
		case popple.KarmaHandler:
			who, err := popple.ParseKarmaArgs(body)
			if err != nil {
				err = s.MessageReactionAdd(m.ChannelID, m.ID, "❓")
				if err != nil {
					log.Println("failed to add reaction to message", err)
				}
				return
			}

			levels, err := action(repo, m.GuildID, who)
			if err != nil {
				log.Println("KarmaHandler", err)
				return
			}

			if len(levels) < 1 {
				return
			}

			var r strings.Builder
			err = levelsTemplate.Execute(&r, levels)
			if err != nil {
				log.Println("levelsTemplate.Execute", err)
				return
			}

			_, err = s.ChannelMessageSend(m.ChannelID, r.String())
			if err != nil {
				log.Println("failed to send message to channel", err)
				return
			}
		case popple.LeaderboardHandler:
			limit, err := popple.ParseLeaderboardArgs(body)
			if errors.Is(err, poperr.ErrInvalidArgument) {
				_, err = s.ChannelMessageSend(m.ChannelID, "The number of entries to list must be a positive non-zero integer")
				if err != nil {
					log.Println("failed to send message to channel", err)
					return
				}
			}

			board, err := action(repo, m.GuildID, limit)
			if err != nil {
				log.Println("LeaderboardHandler", err)
				return
			}

			if len(board) < 1 {
				return
			}

			var r strings.Builder
			err = boardTemplate.Execute(&r, board)
			if err != nil {
				log.Println("boardTemplate.Execute", err)
				return
			}

			_, err = s.ChannelMessageSend(m.ChannelID, r.String())
			if err != nil {
				log.Println("failed to send message to channel", err)
				return
			}
		case popple.LoserboardHandler:
			limit, err := popple.ParseLoserboardArgs(body)
			if errors.Is(err, poperr.ErrInvalidArgument) {
				_, err = s.ChannelMessageSend(m.ChannelID, "The number of entries to list must be a positive non-zero integer")
				if err != nil {
					log.Println("failed to send message to channel", err)
					return
				}
			}

			board, err := action(repo, m.GuildID, limit)
			if err != nil {
				log.Println("LoserboardHandler", err)
				return
			}

			if len(board) < 1 {
				return
			}

			var r strings.Builder
			err = boardTemplate.Execute(&r, board)
			if err != nil {
				log.Println("boardTemplate.Execute", err)
				return
			}

			_, err = s.ChannelMessageSend(m.ChannelID, r.String())
			if err != nil {
				log.Println("failed to send message to channel", err)
				return
			}
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
