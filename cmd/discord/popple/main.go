// Popple counts karma for your Discord server.
package main

import (
	"database/sql"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "modernc.org/sqlite"

	"github.com/connorkuehl/popple"
)

// Database is the SQLite file that Popple will use if one is not provided.
//
// This file is relative to the Popple process's current working directory.
const Database string = "popple.sqlite"

var defaultWorkers uint

func init() {
	numCPU := runtime.NumCPU()
	if numCPU < 1 {
		panic("zero CPUs")
	}

	defaultWorkers = uint(numCPU)
}

type response struct {
	s *discordgo.Session
	m *discordgo.Message
}

func (r response) SendMessageToChannel(msg string) error {
	_, err := r.s.ChannelMessageSend(r.m.ChannelID, msg)
	return err
}

func (r response) SendReply(msg string) error {
	_, err := r.s.ChannelMessageSendReply(r.m.ChannelID, msg, r.m.MessageReference)
	return err
}

func (r response) React(emojiID string) error {
	err := r.s.MessageReactionAdd(r.m.ChannelID, r.m.ID, emojiID)
	return err
}

func main() {
	start := time.Now()
	tokenFile := flag.String("token", "", "path to file containing bot token")
	numWorkers := flag.Uint("workers", defaultWorkers, "Number of worker threads to spawn")
	dbFile := flag.String("db", Database, "Path to database file")
	timeout := flag.Duration("deadline", 3*time.Second, "How long to wait for workers to exit when shutting down")
	flag.Parse()

	if *tokenFile == "" {
		log.Fatalln("Token file must be supplied as a command line argument")
	}

	if *numWorkers < 1 {
		*numWorkers = 1
	}

	db, err := sql.Open("sqlite", *dbFile)
	if err != nil {
		log.Fatalf("Failed to open database: %s\n", err)
	}

	token, err := ioutil.ReadFile(*tokenFile)
	if err != nil {
		log.Fatalf("Failed to read token from %s\n", *tokenFile)
	}

	session, err := discordgo.New("Bot " + string(token))
	if err != nil {
		log.Fatalf("Failed to initialize Discord library: %s\n", err)
	}

	if err := session.Open(); err != nil {
		log.Fatalf("Error connecting to Discord: %s\n", err)
	}
	log.Println("Popple is online")

	cancel := make(chan struct{})
	workQueue := make(chan func(), *numWorkers+1)

	var wg sync.WaitGroup
	wg.Add(int(*numWorkers))

	for i := uint(0); i < *numWorkers; i++ {
		log.Printf("starting worker %d\n", i)
		go worker(workQueue, cancel, &wg)
	}

	app, err := popple.NewApp(db, start)
	if err != nil {
		log.Fatalf("%v", err)
	}
	app.Router.Bot = "@" + session.State.User.Username

	detachMessageCreateHandler := session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// don't process messages sent by the bot
		if s.State.User.ID == m.Author.ID {
			return
		}

		isDM := len(m.GuildID) == 0
		msg := strings.TrimSpace(m.ContentWithMentionsReplaced())

		req := popple.Request{IsDM: isDM, GuildID: m.Message.GuildID, Message: msg}
		rsp := response{s, m.Message}
		workQueue <- func() {
			app.Router.Route(req, rsp)
		}
	})

	sessionChannel := make(chan os.Signal, 1)
	signal.Notify(sessionChannel, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sessionChannel
	log.Println("received signal")

	close(cancel) // workers will read zero-value of this from a closed chan
	detachMessageCreateHandler()
	log.Println("detached handler, sending cancel request to workers")

	deadline := time.After(*timeout)
	workersDone := make(chan struct{})

	go func() {
		wg.Wait()
		close(workersDone)
	}()

	select {
	case <-deadline:
		log.Println("cancellation deadline has passed")
	case <-workersDone:
		log.Println("all workers have acknowledged cancellation")
	}

	session.Close()
}

func worker(workQueue <-chan func(), cancel <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-cancel:
			return
		case cmd := <-workQueue:
			cmd()
		}
	}
}
