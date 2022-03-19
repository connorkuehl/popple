package popple

import (
	"regexp"
)

type Mux struct {
	botName       string
	handlers      map[*regexp.Regexp]interface{}
	onAnnounce    AnnounceHandler
	onBumpKarma   BumpKarmaHandler
	onKarma       KarmaHandler
	onLeaderboard LeaderboardHandler
	onLoserboard  LoserboardHandler
}

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

func (m *Mux) Route(message string) (action interface{}, body string) {
	for matcher, action := range m.handlers {
		if matched := matcher.ReplaceAllString(message, ""); matched != message {
			return action, matched
		}
	}

	return m.onBumpKarma, message
}
