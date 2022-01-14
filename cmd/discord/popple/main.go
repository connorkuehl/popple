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
	"unicode"

	"github.com/bwmarrin/discordgo"
	_ "modernc.org/sqlite"

	"github.com/connorkuehl/popple"
	"github.com/connorkuehl/popple/adapter"

	"github.com/connorkuehl/popple/cmd/discord/popple/internal/cli"
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	tokenFile, err := os.Open(*cli.TokenPath)
	if err != nil {
		return err
	}

	tokenScanner := bufio.NewScanner(tokenFile)
	tokenScanner.Split(bufio.ScanLines)
	if !tokenScanner.Scan() {
		return errors.New("failed to scan token file")
	}

	token := tokenScanner.Text()

	db, err := sql.Open("sqlite", *cli.DatabasePath)
	if err != nil {
		return err
	}
	defer db.Close()

	pl, err := adapter.NewSQLitePersistenceLayer(db)
	if err != nil {
		return err
	}

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return err
	}

	err = session.Open()
	if err != nil {
		return err
	}
	defer session.Close()

	ppl := popple.New(pl)

	detachMessageCreateHandler := session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if s.State.User.ID == m.Author.ID {
			return
		}

		isDM := len(m.GuildID) == 0
		if isDM {
			return
		}

		username := "@" + s.State.User.Username

		content := strings.TrimSpace(m.ContentWithMentionsReplaced())
		body := strings.NewReader(content)

		leaderboard := func(top bool) {
			entries, err := ppl.Leaderboard(m.GuildID, top, body)
			if errors.Is(err, popple.ErrInvalidLimit) {
				_, err := s.ChannelMessageSend(m.ChannelID, "Try a non-zero positive integer next time.")
				if err != nil {
					log.Println(err)
				}
				return
			}

			if err != nil {
				log.Println(err)
				return
			}

			if len(entries) < 1 {
				_, err := s.ChannelMessageSend(m.ChannelID, "No one's bumped any karma yet.")
				if err != nil {
					log.Println(err)
				}
				return
			}

			var sep string
			var reply strings.Builder
			for _, entry := range entries {
				reply.WriteString(fmt.Sprintf("%s* %s has %d karma.", sep, entry.Name, entry.Karma))
				sep = "\n"
			}

			_, err = s.ChannelMessageSend(m.ChannelID, reply.String())
			if err != nil {
				log.Println(err)
				return
			}
		}

		routes := []struct {
			prefix string
			f      func()
		}{
			// set announce
			{fmt.Sprintf("%s announce", username), func() {
				err := ppl.SetAnnounce(m.GuildID, body)
				if errors.Is(err, popple.ErrMissingArgument) ||
					errors.Is(err, popple.ErrInvalidAnnounceSetting) {
					_, err := s.ChannelMessageSend(
						m.ChannelID,
						fmt.Sprintf(
							"The valid options are: %s",
							strings.Join([]string{"\"on\"", "\"off\"", "\"yes\"", "\"no\""}, ", "),
						))
					if err != nil {
						log.Println(err)
					}
					return
				}

				if err != nil {
					log.Println(err)
					return
				}

				err = s.MessageReactionAdd(m.ChannelID, m.ID, "👍")
				if err != nil {
					log.Println(err)
				}
			}},

			// check karma counts
			{fmt.Sprintf("%s karma", username), func() {
				levels, err := ppl.Karma(m.GuildID, body)
				if err != nil {
					log.Println(err)
					return
				}

				var reply strings.Builder
				var sep string
				for w, k := range levels {
					reply.WriteString(fmt.Sprintf("%s%s has %d karma.", sep, w, k))
					sep = " "
				}

				_, err = s.ChannelMessageSend(m.ChannelID, reply.String())
				if err != nil {
					log.Println(err)
				}
			}},

			// check loserboard
			{fmt.Sprintf("%s bot", username), func() {
				leaderboard(false)
			}},

			// check leaderboard
			{fmt.Sprintf("%s top", username), func() {
				leaderboard(true)
			}},
		}

		// try to match the message content to a command
		for _, route := range routes {
			// see if this is a command like "@Popple announce off"
			if withoutCmd := strings.TrimPrefix(content, route.prefix); withoutCmd != content {
				// this is an extra check to make sure the command prefix is separated by a space or
				// by nothing to ensure we don't accidentally accept "@Popple announceoff" (which is wrong)
				// as opposed to "@Popple announce off" (which is correct).
				if c := strings.TrimLeftFunc(withoutCmd, unicode.IsSpace); len(withoutCmd) == 0 || c != withoutCmd {
					body = strings.NewReader(c)
					route.f()
					return
				}
			}
		}

		// otherwise, fall through to just seeing if karma has been bumped
		levels, noAnnounce, err := ppl.BumpKarma(m.GuildID, body)
		if err != nil {
			log.Println(err)
			return
		}

		if noAnnounce {
			return
		}

		if len(levels) < 1 {
			return
		}

		var reply strings.Builder
		var sep string
		for w, k := range levels {
			reply.WriteString(fmt.Sprintf("%s%s has %d karma.", sep, w, k))
			sep = " "
		}

		_, err = s.ChannelMessageSend(m.ChannelID, reply.String())
		if err != nil {
			log.Println(err)
			return
		}
	})
	defer detachMessageCreateHandler()

	signaled := make(chan os.Signal, 1)
	signal.Notify(signaled, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-signaled

	return nil
}
