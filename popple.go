// Popple counts karma for your Discord server.
package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
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

	err = session.Open()
	if err != nil {
		log.Fatalf("Error connecting to Discord: %s\n", err)
	}
	log.Printf("Popple is online, running version %s\n", Version)

	cancel := make(chan struct{})
	workQueue := make(chan func(), *numJobs)

	for i := uint(0); i < *numWorkers; i++ {
		go worker(workQueue, cancel)
	}

	router := router{}
	router.bot = "@" + session.State.User.Username
	router.addRoute("announce", func(req request, rsp responseWriter) {
		SetAnnounce(req, rsp, db)
	})
	router.addRoute("help", func(req request, rsp responseWriter) {
		SendHelp(req, rsp)
	})
	router.addRoute("karma", func(req request, rsp responseWriter) {
		CheckKarma(req, rsp, db)
	})
	router.addRoute("bot", func(req request, rsp responseWriter) {
		Bot(req, rsp, db)
	})
	router.addRoute("top", func(req request, rsp responseWriter) {
		Top(req, rsp, db)
	})
	router.addRoute("version", func(req request, rsp responseWriter) {
		SendVersion(req, rsp)
	})

	// just check for karma operations by default if no other commands
	// were matched
	router.addRoute("*", func(req request, rsp responseWriter) {
		ModKarma(req, rsp, db)
	})

	detachMessageCreateHandler := session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// don't process messages sent by the bot
		if s.State.User.ID == m.Author.ID {
			return
		}

		isDM := len(m.GuildID) == 0
		msg := m.ContentWithMentionsReplaced()

		req := request{isDM: isDM, guildID: m.Message.GuildID, message: msg}
		rsp := response{s, m.Message}
		workQueue <- func() {
			router.route(req, rsp)
		}
	})

	sessionChannel := make(chan os.Signal, 1)
	signal.Notify(sessionChannel, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sessionChannel
	detachMessageCreateHandler()
	close(cancel)
	close(workQueue)

	session.Close()
}

func worker(workQueue <-chan func(), cancel <-chan struct{}) {
	for {
		select {
		case <-cancel:
			return
		case cmd := <-workQueue:
			cmd()
		}
	}
}
