package main

import (
	"fmt"
	"os"
	"strings"

	"gorm.io/gorm"
)

const COMMAND_SIGIL string = "!"

func IsCommand(cmdID string, s string) bool {
	return strings.HasPrefix(s, COMMAND_SIGIL+cmdID)
}

func CheckKarma(job *Job, db *gorm.DB) {
	s := job.Session
	m := job.Message
	guildID := m.GuildID

	words := strings.Fields(m.ContentWithMentionsReplaced())[1:]
	if len(words) < 1 {
		return
	}

	reply := strings.Builder{}

	for _, word := range words {
		name := ParseName(word)

		var entity Entity
		db.Where(&Entity{GuildID: guildID, Name: name}).First(&entity)
		reply.WriteString(fmt.Sprintf("%s has %d karma.", name, entity.Karma))
		reply.WriteString(" ")
	}

	_, err := s.ChannelMessageSend(m.ChannelID, reply.String())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when sending reply to channel: %s\n", err)
	}
}

func ModKarma(job *Job, db *gorm.DB) {
	s := job.Session
	m := job.Message
	guildID := m.GuildID

	words := strings.Fields(m.ContentWithMentionsReplaced())

	updates := make(map[string]Modifier)
	reply := strings.Builder{}

	for _, word := range words {
		mod := ParseModifier(word)
		if mod == nil {
			continue
		}

		if mod.Name == "" {
			fmt.Fprintf(os.Stderr, "Error: parsed an empty name from token %s\n", word)
			continue
		}

		/* Are these copying struct values? */
		prev := updates[mod.Name]
		prev.NetKarma += mod.NetKarma
		prev.Name = mod.Name
		updates[mod.Name] = Modifier{Name: prev.Name, NetKarma: prev.NetKarma}
	}

	processed := 0
	mapLen := len(updates)

	if mapLen < 1 {
		return
	}

	for _, v := range updates {
		var entity Entity

		db.Where(&Entity{GuildID: guildID, Name: v.Name}).FirstOrCreate(&entity)
		entity.Karma += v.NetKarma

		reply.WriteString(fmt.Sprintf("%s has %d karma", entity.Name, entity.Karma))
		if entity.Karma == 0 {
			reply.WriteString(" and has therefore been garbage-collected")
			db.Delete(entity)
		} else {
			db.Save(entity)
		}
		reply.WriteString(".")
		if processed < mapLen-1 {
			reply.WriteString(" ")
		}

		processed++
	}

	_, err := s.ChannelMessageSend(m.ChannelID, reply.String())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when sending reply to channel: %s\n", err)
	}
}
