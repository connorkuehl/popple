package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

type Context struct {
	Job    *Job
	DB     *gorm.DB
	Header string
}

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
		fmt.Fprintf(os.Stderr, "Error when sending reply to channel: %s\n", err)
	}
}

func SetAnnounce(ctx *Context) {
	db := ctx.DB
	m := ctx.Job.Message
	guildID := m.GuildID

	message := m.ContentWithMentionsReplaced()[len(ctx.Header):]

	var on bool
	if strings.HasPrefix(message, "on") || strings.HasPrefix(message, "yes") {
		on = true
	} else if strings.HasPrefix(message, "off") || strings.HasPrefix(message, "no") {
		on = false
	} else {
		// FIXME: I don't know exactly what an "emojiID" is, but I'd like
		// to react with a ? emoji.
		// s.MessageReactionAdd(m.ChannelID, m.ID, "")
		return
	}

	var cfg Config
	db.Where(&Config{GuildID: guildID}).FirstOrCreate(&cfg)
	cfg.NoAnnounce = !on
	db.Save(cfg)

	// FIXME: I don't know what an emojiID is supposed to be, but it'd be cool
	// to react with a thumbs up.
	// s.MessageReactionAdd(m.ChannelID, m.ID, "")
}

func SendHelp(ctx *Context) {
	m := ctx.Job.Message
	reply := "Usage: https://github.com/connorkuehl/popple#usage"

	_, err := ctx.Job.Session.ChannelMessageSend(m.ChannelID, reply)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending message: %s", err)
	}
}

func SendVersion(ctx *Context) {
	m := ctx.Job.Message
	_, err := ctx.Job.Session.ChannelMessageSend(m.ChannelID, fmt.Sprintf("I'm running version %s.", Version))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending version: %s", err)
	}
}

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
			fmt.Fprintf(os.Stderr, "Error when sending reply to channel: %s\n", err)
		}
	}
}

func Top(ctx *Context) {
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
	db.Where(&Entity{GuildID: m.GuildID}).Limit(limit).Order("karma desc").Find(&entities)

	board := strings.Builder{}
	for _, entity := range entities {
		board.WriteString(fmt.Sprintf("* %s (%d karma)\n", entity.Name, entity.Karma))
	}

	_, err := s.ChannelMessageSend(m.ChannelID, board.String())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending message to channel: %s\n", err)
	}
}

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
