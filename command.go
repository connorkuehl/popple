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
	modifiers := ParseModifiers(message)

	reply := strings.Builder{}

	for subject, _ := range modifiers {
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
	_, err := ctx.Job.Session.ChannelMessageSend(m.ChannelID, Version)
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
	modifiers := ParseModifiers(message)

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
	board.WriteString("```")
	for i := 0; i < len(entities); i++ {
		board.WriteString(fmt.Sprintf("%d\t%s\n", entities[i].Karma, entities[i].Name))
	}
	board.WriteString("```")

	if len(entities) == 0 {
		return
	}

	_, err := s.ChannelMessageSend(m.ChannelID, board.String())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending message to channel: %s\n", err)
	}
}
