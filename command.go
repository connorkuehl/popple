package popple

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

	"gorm.io/gorm"
)

// Router routes an incoming message to the appropriate Popple
// message handler.
type Router struct {
	Bot      string
	routes   []route
	catchall route
}

func (r *Router) addRoute(name string, work commandFn) {
	route := route{name, work}
	if name == "*" {
		r.catchall = route
	} else {
		r.routes = append(r.routes, route)
	}
}

// Route invokes the appropriate Popple message handler for the
// Request.
func (r *Router) Route(req Request, rsp ResponseWriter) {
	matchers := []struct {
		prefixer         func(r route) string
		preconditionsMet func(r route) bool
	}{
		{prefixer: func(rt route) string { return fmt.Sprintf("%s %s", r.Bot, rt.match) }, preconditionsMet: func(rt route) bool { return true }},
		{prefixer: func(rt route) string { return rt.match }, preconditionsMet: func(rt route) bool { return req.IsDM }},
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
			if message := strings.TrimPrefix(req.Message, prefix); message != req.Message && matcher.preconditionsMet(route) {
				// ensure the message is an entire command ("@Popple help") or ensure
				// that the command prefix is separated from the rest of the message/
				// arguments by whitespace ("@Popple announce on")
				if trimmed := strings.TrimLeftFunc(message, unicode.IsSpace); len(message) == 0 || trimmed != message {
					req.Message = trimmed
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

// Request represents an incoming message that the Popple bot may
// want to respond to.
type Request struct {
	IsDM    bool
	GuildID string
	Message string
}

// ResponseWriter implementors transmit Popple's responses into
// the respective chat implementation.
type ResponseWriter interface {
	SendMessageToChannel(msg string) error
	SendReply(msg string) error
	React(emoji string) error
}

type commandFn func(req Request, rsp ResponseWriter)

// CheckKarma allows server inhabitants to query karma levels
// for subjects they have incremented or decremented over time.
func CheckKarma(req Request, rsp ResponseWriter, db *gorm.DB) {
	if req.IsDM {
		return
	}

	var sep string

	subjects := marshalSubjects(parseSubjects(req.Message))

	reply := strings.Builder{}

	for subject := range subjects {
		var entty Entity
		db.Where(&Entity{GuildID: req.GuildID, Name: subject}).First(&entty)
		reply.WriteString(fmt.Sprintf("%s%s has %d karma.", sep, subject, entty.Karma))
		sep = " "
	}

	message := strings.TrimSpace(reply.String())
	if len(message) == 0 {
		return
	}

	if err := rsp.SendMessageToChannel(reply.String()); err != nil {
		log.Printf("Error when sending reply to channel: %s\n", err)
	}
}

// SetAnnounce allows server inhabitants to enable or disable Popple
// announcements when karma is modified from a message.
func SetAnnounce(req Request, rsp ResponseWriter, db *gorm.DB) {
	if req.IsDM {
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(req.Message))
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
		if err := rsp.SendReply("Announce settings are: \"yes\", \"no\", \"on\", \"off\""); err != nil {
			log.Printf("Error when sending reply: %v", err)
		}
		return
	}

	var cfg Config
	db.Where(&Config{GuildID: req.GuildID}).FirstOrCreate(&cfg)
	cfg.NoAnnounce = !on
	db.Save(cfg)

	if err := rsp.React("👍"); err != nil {
		log.Printf("Error when sending reply: %v", err)
	}
}

// SendHelp allows server inhabitants to request usage information.
func SendHelp(req Request, rsp ResponseWriter) {
	reply := "Usage: https://github.com/connorkuehl/popple#usage"

	if err := rsp.SendMessageToChannel(reply); err != nil {
		log.Printf("Error sending message: %s", err)
	}
}

// SendVersion allows server inhabitants to see what Popple revision
// is running.
func SendVersion(req Request, rsp ResponseWriter) {
	if err := rsp.SendMessageToChannel(fmt.Sprintf("I'm running version %s.", Version)); err != nil {
		log.Printf("Error sending version: %s", err)
	}
}

// ModKarma is the default Popple action that will be taken when no other
// subcommand is identified in the message.
//
// Popple will scan the entire message, parse out any karma subjects,
// count up the karma, and reply with the karma modifications that the
// message has made resulted in.
func ModKarma(req Request, rsp ResponseWriter, db *gorm.DB) {
	if req.IsDM {
		return
	}

	modifiers := marshalSubjects(parseSubjects(req.Message))

	reply := strings.Builder{}

	var sep string
	for subject, netKarma := range modifiers {
		if netKarma == 0 {
			continue
		}

		var entty Entity

		db.Where(&Entity{GuildID: req.GuildID, Name: subject}).FirstOrCreate(&entty)
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

	var cfg Config
	db.Where(&Config{GuildID: req.GuildID}).FirstOrCreate(&cfg)

	if !cfg.NoAnnounce {
		if err := rsp.SendMessageToChannel(reply.String()); err != nil {
			log.Printf("Error when sending reply to channel: %s\n", err)
		}
	}
}

// Bot allows server inhabitants to see who is "in the lead" for
// the LEAST amount of karma.
func Bot(req Request, rsp ResponseWriter, db *gorm.DB) {
	board(req, rsp, db, "asc")
}

// Top allows server inhabitants to see who is in the lead in terms
// of karma accumulated.
func Top(req Request, rsp ResponseWriter, db *gorm.DB) {
	board(req, rsp, db, "desc")
}

// Uptime creates a formatted string informing a user of the time since
// the last crash
func Uptime(req Request, rsp ResponseWriter, start time.Time) {
	uptime := time.Since(start).Truncate(time.Second)
	if err := rsp.SendReply(fmt.Sprintf("It has been %s since my last crash.", uptime)); err != nil {
		log.Printf("Error when sending reply: %v", err)
	}
}

func board(req Request, rsp ResponseWriter, db *gorm.DB, sort string) {
	if req.IsDM {
		return
	}

	limit := 10

	message := req.Message
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

	var entities []Entity
	db.Where(&Entity{GuildID: req.GuildID}).Order(fmt.Sprintf("karma %s", sort)).Limit(limit).Find(&entities)

	board := strings.Builder{}
	for _, entity := range entities {
		board.WriteString(formatKarmaLeaderboardEntry(entity.Name, entity.Karma))
	}

	reply := board.String()
	if len(reply) == 0 {
		return
	}

	if err := rsp.SendMessageToChannel(board.String()); err != nil {
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
