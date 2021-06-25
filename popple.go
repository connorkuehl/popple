// Popple counts karma for your Discord server.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database is the SQLite file that Popple will use if one is not provided.
//
// This file is relative to the Popple process's current working directory.
const Database string = "popple.sqlite"

// DefaultWorkers is the default number of goroutines that will be spun up
// if not overridden on the command line.
const DefaultWorkers uint = 4

// DefaultJobs is the maximum number of pending jobs that will be held before
// new jobs start getting dropped.
//
// As far as Popple is concerned, a "job" is a pending Discord message that
// Popple must process to determine how it will act (or not act).
const DefaultJobs uint = 128

func main() {
	tokenFile := flag.String("token", "", "path to file containing bot token")
	numWorkers := flag.Uint("workers", DefaultWorkers, "Number of worker threads to spawn")
	dbFile := flag.String("db", Database, "Path to database file")
	numJobs := flag.Uint("jobs", DefaultJobs, "Maximum queue size for jobs")
	flag.Parse()

	if *tokenFile == "" {
		log.Fatalln("Token file must be supplied as a command line argument")
	}

	if *numWorkers < 1 {
		*numWorkers = 1
	}

	db, err := gorm.Open(sqlite.Open(*dbFile), &gorm.Config{
		/* FIXME: Might want to tweak this some more. I turned it off because
		it would log when an entry is not found and that's fine, especially for
		the !karma command. */
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("Failed to open database: %s\n", err)
	}

	token, err := ioutil.ReadFile(*tokenFile)
	if err != nil {
		log.Fatalf("Failed to read token from %s\n", *tokenFile)
	}

	db.AutoMigrate(&Config{})
	db.AutoMigrate(&Entity{})

	session, err := discordgo.New("Bot " + string(token))
	if err != nil {
		log.Fatalf("Failed to initialize Discord library: %s\n", err)
	}

	cancel := make(chan struct{})
	workQueue := make(chan commandFn, *numJobs)

	for i := uint(0); i < *numWorkers; i++ {
		go worker(workQueue, cancel)
	}

	// Generate the command dispatch table up front.
	//
	// These are all closures to capture the state from this function,
	// such as the GORM database (db).
	//
	// This is useful because the workQueue can simply be a workqueue
	// of func() and we can supply all the relevent data items as needed
	// from the closure capture.
	cmds := []struct {
		verb    string
		hasArgs bool
		command func(poppleRequest, poppleResponse) commandFn
	}{
		{"announce", true, func(req poppleRequest, rsp poppleResponse) commandFn {
			return func() {
				SetAnnounce(req, rsp, db)
			}
		}},
		{"help", false, func(req poppleRequest, rsp poppleResponse) commandFn {
			return func() {
				SendHelp(req, rsp)
			}
		}},
		{"karma", true, func(req poppleRequest, rsp poppleResponse) commandFn {
			return func() {
				CheckKarma(req, rsp, db)
			}
		}},
		{"bot", false, func(req poppleRequest, rsp poppleResponse) commandFn {
			return func() {
				Bot(req, rsp, db)
			}
		}},
		{"top", false, func(req poppleRequest, rsp poppleResponse) commandFn {
			return func() {
				Top(req, rsp, db)
			}
		}},
		{"version", false, func(req poppleRequest, rsp poppleResponse) commandFn {
			return func() {
				SendVersion(req, rsp)
			}
		}},
	}

	session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// don't process messages sent by the bot
		if s.State.User.ID == m.Author.ID {
			return
		}

		isDM := len(m.GuildID) == 0
		bot := "@" + s.State.User.Username

		msg := m.ContentWithMentionsReplaced()
		for _, c := range cmds {
			var spacer string
			if c.hasArgs {
				spacer = " "
			}

			var strip string
			fullPrefix := fmt.Sprintf("%s %s%s", bot, c.verb, spacer)
			dmPrefix := fmt.Sprintf("%s%s", c.verb, spacer)
			if strings.HasPrefix(msg, fullPrefix) {
				strip = fullPrefix
			} else if isDM && strings.HasPrefix(msg, dmPrefix) {
				strip = dmPrefix
			} else {
				// message doesn't appear to be addressing the bot with a command
				// just move on
				continue
			}

			// remove the @Bot (if it's there) as well as the command text so
			// the command processing layer doesn't need to worry about it
			msg = msg[len(strip):]

			workQueue <- c.command(message{m.Message, msg}, response{s, m.Message})
			return
		}

		// default action is to just check for karma operations
		workQueue <- func() {
			ModKarma(message{m.Message, msg}, response{s, m.Message}, db)
		}
	})

	err = session.Open()
	if err != nil {
		/* Should these be `defer`red? */
		close(cancel)
		close(workQueue)
		log.Fatalf("Error connecting to Discord: %s\n", err)
	}

	log.Printf("Popple is online, running version %s\n", Version)

	sessionChannel := make(chan os.Signal, 1)
	signal.Notify(sessionChannel, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sessionChannel
	close(cancel)
	close(workQueue)

	session.Close()
}

func worker(workQueue <-chan commandFn, cancel <-chan struct{}) {
	for {
		select {
		case <-cancel:
			return
		case cmd := <-workQueue:
			cmd()
		}
	}
}
