package main

import (
	"fmt"
	"os"
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

	_, err := s.ChannelMessageSend(m.ChannelID, reply.String())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when sending reply to channel: %s\n", err)
	}
}
