package main

// Module job handles the "bottom half" style of message processing
// for the Popple bot.
//
// The underlying Discord library, discordgo, invokes a callback
// which places each incoming message into a "Job" on a work queue
// which is shared by many worker goroutines.
//
// This allows the handler to return immediately and all messages
// can be processed asynchronously without blocking anything else.

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

// Job represents an incoming message that the bot must process
// but it also bundles up the necessary bot and session state for
// the bot to perform its actions.
type Job struct {
	Session *discordgo.Session
	Message *discordgo.MessageCreate
}

// doWork scans a Discord message enclosed in a Job object and
// dispatches it to the relevant business-logic functions based
// on the content of the message (i.e., if it is a bot subcommand
// or if it's simply a message that Popple must read to check for
// karma events).
func doWork(job *Job, db *gorm.DB) {
	if job.Session.State.User.ID == job.Message.Author.ID {
		return
	}

	myUser := "@" + job.Session.State.User.Username
	cmds := []struct {
		verb    string
		call    func(*Context)
		hasArgs bool
	}{
		{"announce", SetAnnounce, true},
		{"help", SendHelp, false},
		{"karma", CheckKarma, true},
		{"bot", Bot, false},
		{"top", Top, false},
		{"version", SendVersion, false},
	}

	msg := job.Message.ContentWithMentionsReplaced()
	for _, c := range cmds {
		spacer := ""
		if c.hasArgs {
			spacer = " "
		}

		header := fmt.Sprintf("%s %s%s", myUser, c.verb, spacer)
		if strings.HasPrefix(msg, header) {
			c.call(&Context{job, db, header})
			return
		}
	}

	ModKarma(&Context{job, db, ""})
}

// worker is the goroutine entry point and idling place for
// worker goroutines.
func worker(workQueue <-chan Job, cancel <-chan struct{}, db *gorm.DB) {
	for {
		select {
		case <-cancel:
			return
		case job := <-workQueue:
			doWork(&job, db)
		}
	}
}
