package main

import (
	"bufio"
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
	poperr "github.com/connorkuehl/popple/errors"
	sqlite_repo "github.com/connorkuehl/popple/repo/sqlite"
)

var (
	database  = kingpin.Flag("database", "Path to the SQLite database.").Required().String()
	tokenFile = kingpin.Flag("token", "Path to the Discord bot token file.").Required().ExistingFile()
)

var (
	levelsTemplate = template.Must(template.New("levels").Parse(`{{ range $name, $karma := . }}{{ $name }} has {{ $karma }} karma. {{ end }}`))
	boardTemplate  = template.Must(template.New("board").Parse(
		`{{ range $entry := . }}* {{ $entry.Name }} has {{ $entry.Karma }}.
		{{ end }}`))
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	kingpin.Parse()

	f, err := os.OpenFile(*tokenFile, os.O_CREATE|os.O_RDWR, 0o0644)
	if err != nil {
		return fmt.Errorf("failed to open token file: %w", err)
	}
	defer f.Close()

	tokenScanner := bufio.NewScanner(f)
	tokenScanner.Split(bufio.ScanLines)
	if !tokenScanner.Scan() {
		return fmt.Errorf("failed to read token from token file: %w", tokenScanner.Err())
	}

	token := tokenScanner.Text()

	db, err := sql.Open("sqlite", *database)
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

	repoMu := make(chan struct{}, 1)
	repo := sqlite_repo.NewRepository(db)

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
					log.Println("failed to send message: %w", err)
				}
				return
			}
			if err != nil {
				log.Println("ParseAnnounceArgs: %w", err)
				return
			}

			err = action(repo, m.GuildID, on)
			if err != nil {
				log.Println("AnnounceHandler: %w", err)
				return
			}

			err = s.MessageReactionAdd(m.ChannelID, m.ID, "✅")
			if err != nil {
				log.Println("failed to add reaction to message: %w", err)
				return
			}
		case popple.BumpKarmaHandler:
			increments, _ := popple.ParseBumpKarmaArgs(body)

			levels, err := action(repo, m.GuildID, increments)
			if err != nil {
				log.Println("BumpKarmaHandler: %w", err)
				return
			}

			config, err := repo.Config(m.GuildID)
			if errors.Is(err, poperr.ErrNotFound) {
				err = nil
			}

			if err != nil {
				log.Println("repo.Config: %w", err)
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
				log.Println("levelsTemplate.Execute: %w", err)
				return
			}

			_, err = s.ChannelMessageSend(m.ChannelID, r.String())
			if err != nil {
				log.Println("failed to send message to channel: %w", err)
				return
			}
		case popple.KarmaHandler:
			who, err := popple.ParseKarmaArgs(body)
			if err != nil {
				err = s.MessageReactionAdd(m.ChannelID, m.ID, "❓")
				if err != nil {
					log.Println("failed to add reaction to message: %w", err)
				}
				return
			}

			levels, err := action(repo, m.GuildID, who)
			if err != nil {
				log.Println("KarmaHandler: %w", err)
				return
			}

			if len(levels) < 1 {
				return
			}

			var r strings.Builder
			err = levelsTemplate.Execute(&r, levels)
			if err != nil {
				log.Println("levelsTemplate.Execute: %w", err)
				return
			}

			_, err = s.ChannelMessageSend(m.ChannelID, r.String())
			if err != nil {
				log.Println("failed to send message to channel: %w", err)
				return
			}
		case popple.LeaderboardHandler:
			limit, err := popple.ParseLeaderboardArgs(body)
			if errors.Is(err, poperr.ErrInvalidArgument) {
				_, err = s.ChannelMessageSend(m.ChannelID, "The number of entries to list must be a positive non-zero integer")
				if err != nil {
					log.Println("failed to send message to channel: %w", err)
					return
				}
			}

			board, err := action(repo, m.GuildID, limit)
			if err != nil {
				log.Println("LeaderboardHandler: %w", err)
				return
			}

			if len(board) < 1 {
				return
			}

			var r strings.Builder
			err = boardTemplate.Execute(&r, board)
			if err != nil {
				log.Println("boardTemplate.Execute: %w", err)
				return
			}

			_, err = s.ChannelMessageSend(m.ChannelID, r.String())
			if err != nil {
				log.Println("failed to send message to channel: %w", err)
				return
			}
		case popple.LoserboardHandler:
			limit, err := popple.ParseLoserboardArgs(body)
			if errors.Is(err, poperr.ErrInvalidArgument) {
				_, err = s.ChannelMessageSend(m.ChannelID, "The number of entries to list must be a positive non-zero integer")
				if err != nil {
					log.Println("failed to send message to channel: %w", err)
					return
				}
			}

			board, err := action(repo, m.GuildID, limit)
			if err != nil {
				log.Println("LoserboardHandler: %w", err)
				return
			}

			if len(board) < 1 {
				return
			}

			var r strings.Builder
			err = boardTemplate.Execute(&r, board)
			if err != nil {
				log.Println("boardTemplate.Execute: %w", err)
				return
			}

			_, err = s.ChannelMessageSend(m.ChannelID, r.String())
			if err != nil {
				log.Println("failed to send message to channel: %w", err)
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
