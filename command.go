package main

// Module command contains all of the business logic associated
// with the Popple bot and its behaviors.
//
// The routing information for what text commands are wired to
// which function are currently represented in the command
// dispatch table in job.go

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// Context represents all of the necessary state and session
// functionality for the Popple bot to perform its commands
// and to interact with the Discord channels that it is in.
type Context struct {
	Job    *Job
	DB     *gorm.DB
	Header string
}

// CheckKarma allows server inhabitants to query karma levels
// for subjects they have incremented or decremented over time.
func CheckKarma(ctx *Context) {
	var sep string
	db := ctx.DB
	s := ctx.Job.Session
	m := ctx.Job.Message
	guildID := m.GuildID

	message := m.ContentWithMentionsReplaced()[len(ctx.Header):]
	subjects := marshalSubjects(ParseSubjects(message))

	reply := strings.Builder{}

	for subject, _ := range subjects {
		var entity Entity
		db.Where(&Entity{GuildID: guildID, Name: subject}).First(&entity)
		reply.WriteString(fmt.Sprintf("%s%s has %d karma.", sep, subject, entity.Karma))
		sep = " "
	}

	_, err := s.ChannelMessageSend(m.ChannelID, reply.String())
	if err != nil {
		log.Printf("Error when sending reply to channel: %s\n", err)
	}
}

// SetAnnounce allows server inhabitants to enable or disable Popple
// announcements when karma is modified from a message.
func SetAnnounce(ctx *Context) {
	db := ctx.DB
	s := ctx.Job.Session
	m := ctx.Job.Message
	guildID := m.GuildID

	message := m.ContentWithMentionsReplaced()[len(ctx.Header):]

	var on bool
	if strings.HasPrefix(message, "on") || strings.HasPrefix(message, "yes") {
		on = true
	} else if strings.HasPrefix(message, "off") || strings.HasPrefix(message, "no") {
		on = false
	} else {
		_, err := s.ChannelMessageSendReply(m.ChannelID, "Announce settings are: \"yes\", \"no\", \"on\", \"off\"", m.MessageReference)
		if err != nil {
			log.Printf("Error when sending reply: %v", err)
		}
		return
	}

	var cfg Config
	db.Where(&Config{GuildID: guildID}).FirstOrCreate(&cfg)
	cfg.NoAnnounce = !on
	db.Save(cfg)

	err := s.MessageReactionAdd(m.ChannelID, m.ID, "👍")
	if err != nil {
		log.Printf("Error when sending reply: %v", err)
	}
}

// SendHelp allows server inhabitants to request usage information.
func SendHelp(ctx *Context) {
	m := ctx.Job.Message
	reply := "Usage: https://github.com/connorkuehl/popple#usage"

	_, err := ctx.Job.Session.ChannelMessageSend(m.ChannelID, reply)
	if err != nil {
		log.Printf("Error sending message: %s", err)
	}
}

// SendVersion allows server inhabitants to see what Popple revision
// is running.
func SendVersion(ctx *Context) {
	m := ctx.Job.Message
	_, err := ctx.Job.Session.ChannelMessageSend(m.ChannelID, fmt.Sprintf("I'm running version %s.", Version))
	if err != nil {
		log.Printf("Error sending version: %s", err)
	}
}

// ModKarma is the default Popple action that will be taken when no other
// subcommand is identified in the message.
//
// Popple will scan the entire message, parse out any karma subjects,
// count up the karma, and reply with the karma modifications that the
// message has made resulted in.
func ModKarma(ctx *Context) {
	db := ctx.DB
	s := ctx.Job.Session
	m := ctx.Job.Message
	guildID := m.GuildID

	message := m.ContentWithMentionsReplaced()
	modifiers := marshalSubjects(ParseSubjects(message))

	reply := strings.Builder{}

	var sep string
	for subject, netKarma := range modifiers {
		if netKarma == 0 {
			continue
		}

		var entity Entity

		db.Where(&Entity{GuildID: guildID, Name: subject}).FirstOrCreate(&entity)
		entity.Karma += netKarma

		reply.WriteString(fmt.Sprintf("%s%s has %d karma.", sep, entity.Name, entity.Karma))
		if entity.Karma == 0 {
			db.Delete(entity)
		} else {
			db.Save(entity)
		}
		sep = " "
	}

	if reply.Len() == 0 {
		return
	}

	var cfg Config
	db.Where(&Config{GuildID: guildID}).FirstOrCreate(&cfg)

	if !cfg.NoAnnounce {
		_, err := s.ChannelMessageSend(m.ChannelID, reply.String())
		if err != nil {
			log.Printf("Error when sending reply to channel: %s\n", err)
		}
	}
}

// Bot allows server inhabitants to see who is "in the lead" for
// the LEAST amount of karma.
func Bot(ctx *Context) {
	board(ctx, "asc")
}

// Top allows server inhabitants to see who is in the lead in terms
// of karma accumulated.
func Top(ctx *Context) {
	board(ctx, "desc")
}

func board(ctx *Context, sort string) {
	s := ctx.Job.Session
	m := ctx.Job.Message
	db := ctx.DB

	limit := 10

	message := ctx.Job.Message.ContentWithMentionsReplaced()[len(ctx.Header):]
	parts := strings.Fields(message)
	if len(parts) > 0 {
		limitArg, err := strconv.Atoi(parts[0])
		if err == nil && limitArg > 0 {
			limit = limitArg
		}
	}

	var entities []Entity
	db.Where(&Entity{GuildID: m.GuildID}).Order(fmt.Sprintf("karma %s", sort)).Limit(limit).Find(&entities)

	board := strings.Builder{}
	for _, entity := range entities {
		board.WriteString(fmt.Sprintf("* %s (%d karma)\n", entity.Name, entity.Karma))
	}

	_, err := s.ChannelMessageSend(m.ChannelID, board.String())
	if err != nil {
		log.Printf("Error sending message to channel: %s\n", err)
	}
}

// marshalSubjects deduplicates the list of Subjects that ParseSubjects
// returns.
func marshalSubjects(subs []Subject) map[string]int {
	subMap := make(map[string]int)
	for _, s := range subs {
		/* associate @user with user */
		name := s.Name
		if len(name) > 1 && name[0] == '@' {
			name = name[1:]
		}
		karma := subMap[name]
		karma += s.Karma
		subMap[name] = karma
	}

	return subMap
}
