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

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

type poppleRequest interface {
	IsDM() bool
	GuildID() string
	String() string
}

type message struct {
	internal *discordgo.Message
	msg      string
}

func (m message) IsDM() bool {
	return len(m.GuildID()) == 0
}

func (m message) GuildID() string {
	return m.internal.GuildID
}

func (m message) String() string {
	return m.msg
}

type poppleResponse interface {
	SendMessageToChannel(msg string) error
	SendReply(msg string) error
	React(emoji string) error
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

type commandFn func()

// CheckKarma allows server inhabitants to query karma levels
// for subjects they have incremented or decremented over time.
func CheckKarma(req poppleRequest, rsp poppleResponse, db *gorm.DB) {
	if req.IsDM() {
		return
	}

	var sep string

	subjects := marshalSubjects(ParseSubjects(req.String()))

	reply := strings.Builder{}

	for subject, _ := range subjects {
		var entity Entity
		db.Where(&Entity{GuildID: req.GuildID(), Name: subject}).First(&entity)
		reply.WriteString(fmt.Sprintf("%s%s has %d karma.", sep, subject, entity.Karma))
		sep = " "
	}

	err := rsp.SendMessageToChannel(reply.String())
	if err != nil {
		log.Printf("Error when sending reply to channel: %s\n", err)
	}
}

// SetAnnounce allows server inhabitants to enable or disable Popple
// announcements when karma is modified from a message.
func SetAnnounce(req poppleRequest, rsp poppleResponse, db *gorm.DB) {
	if req.IsDM() {
		return
	}

	message := req.String()

	var on bool
	if strings.HasPrefix(message, "on") || strings.HasPrefix(message, "yes") {
		on = true
	} else if strings.HasPrefix(message, "off") || strings.HasPrefix(message, "no") {
		on = false
	} else {
		err := rsp.SendReply("Announce settings are: \"yes\", \"no\", \"on\", \"off\"")
		if err != nil {
			log.Printf("Error when sending reply: %v", err)
		}
		return
	}

	var cfg Config
	db.Where(&Config{GuildID: req.GuildID()}).FirstOrCreate(&cfg)
	cfg.NoAnnounce = !on
	db.Save(cfg)

	err := rsp.React("👍")
	if err != nil {
		log.Printf("Error when sending reply: %v", err)
	}
}

// SendHelp allows server inhabitants to request usage information.
func SendHelp(req poppleRequest, rsp poppleResponse) {
	reply := "Usage: https://github.com/connorkuehl/popple#usage"

	err := rsp.SendMessageToChannel(reply)
	if err != nil {
		log.Printf("Error sending message: %s", err)
	}
}

// SendVersion allows server inhabitants to see what Popple revision
// is running.
func SendVersion(req poppleRequest, rsp poppleResponse) {
	err := rsp.SendMessageToChannel(fmt.Sprintf("I'm running version %s.", Version))
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
func ModKarma(req poppleRequest, rsp poppleResponse, db *gorm.DB) {
	modifiers := marshalSubjects(ParseSubjects(req.String()))

	reply := strings.Builder{}

	var sep string
	for subject, netKarma := range modifiers {
		if netKarma == 0 {
			continue
		}

		var entity Entity

		db.Where(&Entity{GuildID: req.GuildID(), Name: subject}).FirstOrCreate(&entity)
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
	db.Where(&Config{GuildID: req.GuildID()}).FirstOrCreate(&cfg)

	if !cfg.NoAnnounce {
		err := rsp.SendMessageToChannel(reply.String())
		if err != nil {
			log.Printf("Error when sending reply to channel: %s\n", err)
		}
	}
}

// Bot allows server inhabitants to see who is "in the lead" for
// the LEAST amount of karma.
func Bot(req poppleRequest, rsp poppleResponse, db *gorm.DB) {
	board(req, rsp, db, "asc")
}

// Top allows server inhabitants to see who is in the lead in terms
// of karma accumulated.
func Top(req poppleRequest, rsp poppleResponse, db *gorm.DB) {
	board(req, rsp, db, "desc")
}

func board(req poppleRequest, rsp poppleResponse, db *gorm.DB, sort string) {
	if req.IsDM() {
		return
	}

	limit := 10

	message := req.String()
	parts := strings.Fields(message)
	if len(parts) > 0 {
		limitArg, err := strconv.Atoi(parts[0])
		if err == nil && limitArg > 0 {
			limit = limitArg
		}
	}

	var entities []Entity
	db.Where(&Entity{GuildID: req.GuildID()}).Order(fmt.Sprintf("karma %s", sort)).Limit(limit).Find(&entities)

	board := strings.Builder{}
	for _, entity := range entities {
		board.WriteString(fmt.Sprintf("* %s (%d karma)\n", entity.Name, entity.Karma))
	}

	err := rsp.SendMessageToChannel(board.String())
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
