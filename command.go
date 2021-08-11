package main

// Module command contains all of the business logic associated
// with the Popple bot and its behaviors.
//
// The routing information for what text commands are wired to
// which function are currently represented in the command
// dispatch table in job.go

import (
	"bufio"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

type router struct {
	bot      string
	routes   []route
	catchall route
}

func (r *router) addRoute(name string, work commandFn) {
	route := route{name, work}
	if name == "*" {
		r.catchall = route
	} else {
		r.routes = append(r.routes, route)
	}
}

func (r *router) route(req request, rsp responseWriter) {
	matchers := []struct {
		prefixer         func(r route) string
		preconditionsMet func(r route) bool
	}{
		{prefixer: func(rt route) string { return fmt.Sprintf("%s %s", r.bot, rt.match) }, preconditionsMet: func(rt route) bool { return true }},
		{prefixer: func(rt route) string { return rt.match }, preconditionsMet: func(rt route) bool { return req.isDM }},
	}

	for _, route := range r.routes {
		for _, matcher := range matchers {
			prefix := matcher.prefixer(route)
			// ensure the command verb is its own word
			//		ok: "@Popple help"
			//		no: "@Popple helpasdf"
			//		ok: "@Popple announce on"
			//		no: "@Popple announceon"
			//
			// previously, only the prefix was checked, meaning "@Popple helpasdf"
			// would route to "@Popple help" and "asdf" was passed on as the message
			// body/arguments
			if message := strings.TrimPrefix(req.message, prefix); message != req.message && matcher.preconditionsMet(route) {
				// ensure the message is an entire command ("@Popple help") or ensure
				// that the command prefix is separated from the rest of the message/
				// arguments by whitespace ("@Popple announce on")
				if trimmed := strings.TrimLeftFunc(message, unicode.IsSpace); len(message) == 0 || trimmed != message {
					req.message = trimmed
					route.cmd(req, rsp)
					return
				}
			}
		}
	}

	if r.catchall.cmd != nil {
		r.catchall.cmd(req, rsp)
	}
}

type route struct {
	match string
	cmd   commandFn
}

type request struct {
	isDM    bool
	guildID string
	message string
}

type responseWriter interface {
	sendMessageToChannel(msg string) error
	sendReply(msg string) error
	react(emoji string) error
}

type response struct {
	s *discordgo.Session
	m *discordgo.Message
}

func (r response) sendMessageToChannel(msg string) error {
	_, err := r.s.ChannelMessageSend(r.m.ChannelID, msg)
	return err
}

func (r response) sendReply(msg string) error {
	_, err := r.s.ChannelMessageSendReply(r.m.ChannelID, msg, r.m.MessageReference)
	return err
}

func (r response) react(emojiID string) error {
	err := r.s.MessageReactionAdd(r.m.ChannelID, r.m.ID, emojiID)
	return err
}

type commandFn func(req request, rsp responseWriter)

// checkKarma allows server inhabitants to query karma levels
// for subjects they have incremented or decremented over time.
func checkKarma(req request, rsp responseWriter, db *gorm.DB) {
	if req.isDM {
		return
	}

	var sep string

	subjects := marshalSubjects(ParseSubjects(req.message))

	reply := strings.Builder{}

	for subject := range subjects {
		var entty entity
		db.Where(&entity{GuildID: req.guildID, Name: subject}).First(&entty)
		reply.WriteString(fmt.Sprintf("%s%s has %d karma.", sep, subject, entty.Karma))
		sep = " "
	}

	message := strings.TrimSpace(reply.String())
	if len(message) == 0 {
		return
	}

	if err := rsp.sendMessageToChannel(reply.String()); err != nil {
		log.Printf("Error when sending reply to channel: %s\n", err)
	}
}

// setAnnounce allows server inhabitants to enable or disable Popple
// announcements when karma is modified from a message.
func setAnnounce(req request, rsp responseWriter, db *gorm.DB) {
	if req.isDM {
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(req.message))
	scanner.Split(bufio.ScanWords)
	_ = scanner.Scan()
	setting := scanner.Text()

	var on bool
	switch {
	case setting == "on" || setting == "yes":
		on = true
	case setting == "off" || setting == "no":
		on = false
	default:
		if err := rsp.sendReply("Announce settings are: \"yes\", \"no\", \"on\", \"off\""); err != nil {
			log.Printf("Error when sending reply: %v", err)
		}
		return
	}

	var cfg config
	db.Where(&config{guildID: req.guildID}).FirstOrCreate(&cfg)
	cfg.NoAnnounce = !on
	db.Save(cfg)

	if err := rsp.react("👍"); err != nil {
		log.Printf("Error when sending reply: %v", err)
	}
}

// sendHelp allows server inhabitants to request usage information.
func sendHelp(req request, rsp responseWriter) {
	reply := "Usage: https://github.com/connorkuehl/popple#usage"

	if err := rsp.sendMessageToChannel(reply); err != nil {
		log.Printf("Error sending message: %s", err)
	}
}

// sendVersion allows server inhabitants to see what Popple revision
// is running.
func sendVersion(req request, rsp responseWriter) {
	if err := rsp.sendMessageToChannel(fmt.Sprintf("I'm running version %s.", Version)); err != nil {
		log.Printf("Error sending version: %s", err)
	}
}

// modKarma is the default Popple action that will be taken when no other
// subcommand is identified in the message.
//
// Popple will scan the entire message, parse out any karma subjects,
// count up the karma, and reply with the karma modifications that the
// message has made resulted in.
func modKarma(req request, rsp responseWriter, db *gorm.DB) {
	if req.isDM {
		return
	}

	modifiers := marshalSubjects(ParseSubjects(req.message))

	reply := strings.Builder{}

	var sep string
	for subject, netKarma := range modifiers {
		if netKarma == 0 {
			continue
		}

		var entty entity

		db.Where(&entity{GuildID: req.guildID, Name: subject}).FirstOrCreate(&entty)
		entty.Karma += netKarma

		reply.WriteString(fmt.Sprintf("%s%s.", sep, formatKarmaStatement(entty.Name, entty.Karma)))
		if entty.Karma == 0 {
			db.Delete(entty)
		} else {
			db.Save(entty)
		}
		sep = " "
	}

	if reply.Len() == 0 {
		return
	}

	var cfg config
	db.Where(&config{guildID: req.guildID}).FirstOrCreate(&cfg)

	if !cfg.NoAnnounce {
		if err := rsp.sendMessageToChannel(reply.String()); err != nil {
			log.Printf("Error when sending reply to channel: %s\n", err)
		}
	}
}

// bot allows server inhabitants to see who is "in the lead" for
// the LEAST amount of karma.
func bot(req request, rsp responseWriter, db *gorm.DB) {
	board(req, rsp, db, "asc")
}

// top allows server inhabitants to see who is in the lead in terms
// of karma accumulated.
func top(req request, rsp responseWriter, db *gorm.DB) {
	board(req, rsp, db, "desc")
}

// uptime creates a formatted string informing a user of the time since
// the last crash
func uptime(req request, rsp responseWriter, start time.Time) {
	uptime := time.Since(start).Truncate(time.Second)
	if err := rsp.sendReply(fmt.Sprintf("It has been %s since my last crash.", uptime)); err != nil {
		log.Printf("Error when sending reply: %v", err)
	}
}

func board(req request, rsp responseWriter, db *gorm.DB, sort string) {
	if req.isDM {
		return
	}

	limit := 10

	message := req.message
	parts := strings.Fields(message)
	if len(parts) > 0 {
		limitArg, err := strconv.Atoi(parts[0])
		if err == nil {
			limit = limitArg
		}
	}

	if limit < 1 {
		return
	}

	var entities []entity
	db.Where(&entity{GuildID: req.guildID}).Order(fmt.Sprintf("karma %s", sort)).Limit(limit).Find(&entities)

	board := strings.Builder{}
	for _, entity := range entities {
		board.WriteString(formatKarmaLeaderboardEntry(entity.Name, entity.Karma))
	}

	reply := board.String()
	if len(reply) == 0 {
		return
	}

	if err := rsp.sendMessageToChannel(board.String()); err != nil {
		log.Printf("Error sending message to channel: %s\n", err)
	}
}

func formatKarmaStatement(who string, karma int) string {
	return fmt.Sprintf("%s has %d karma", who, karma)
}

func formatKarmaLeaderboardEntry(who string, karma int) string {
	return fmt.Sprintf("* %s (%d karma)\n", who, karma)
}

// marshalSubjects deduplicates the list of Subjects that ParseSubjects
// returns.
func marshalSubjects(subs []subject) map[string]int {
	subMap := make(map[string]int)
	for _, s := range subs {
		name := s.name
		karma := subMap[name]
		karma += s.karma
		subMap[name] = karma
	}

	return subMap
}
