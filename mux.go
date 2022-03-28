package popple

import (
	"regexp"
)

// Mux routes a message based on its message content to one of the bot's
// handlers.
type Mux struct {
	botName       string
	handlers      map[*regexp.Regexp]interface{}
	onAnnounce    AnnounceHandler
	onBumpKarma   BumpKarmaHandler
	onKarma       KarmaHandler
	onLeaderboard LeaderboardHandler
	onLoserboard  LoserboardHandler
}

// NewMux constructs a Mux with the bot's addressable name (the name is
// significant because it is expected that the message will be prefixed
// with the bot's name if it is a command.)
func NewMux(name string) *Mux {
	m := &Mux{
		botName:       name,
		handlers:      make(map[*regexp.Regexp]interface{}),
		onAnnounce:    Announce,
		onBumpKarma:   BumpKarma,
		onKarma:       Karma,
		onLeaderboard: Leaderboard,
		onLoserboard:  Loserboard,
	}

	handlers := map[string]interface{}{
		"announce": m.onAnnounce,
		"karma":    m.onKarma,
		"top":      m.onLeaderboard,
		"bot":      m.onLoserboard,
	}

	// install handlers
	// the prefix requires the message to be prefaced with the bot's name
	prefix := `^(` + m.botName + `)`
	for cmd, cb := range handlers {
		m.handlers[regexp.MustCompile(prefix+" "+cmd)] = cb
	}

	return m
}

// Route takes a message and returns a handler type based on its content (if
// the message is routed to a specific bot command, the command will be stripped
// from the message in the returned 'body', leaving only the arguments to the
// command in the 'body'; else, the original message is returned with the default
//handler type.)
func (m *Mux) Route(message string) (action interface{}, body string) {
	for matcher, action := range m.handlers {
		if matched := matcher.ReplaceAllString(message, ""); matched != message {
			return action, matched
		}
	}

	return m.onBumpKarma, message
}
