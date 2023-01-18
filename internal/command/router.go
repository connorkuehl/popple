package command

import "regexp"

type ArgParser interface {
	ParseArg(s string) error
}

type ArgConstructor func() ArgParser

type Router struct {
	name     string
	handlers map[*regexp.Regexp]ArgConstructor
}

func NewRouter(name string) *Router {
	r := Router{
		name:     name,
		handlers: make(map[*regexp.Regexp]ArgConstructor),
	}

	handlers := map[string]ArgConstructor{
		"announce": func() ArgParser { return new(SetAnnounceArgs) },
		"karma":    func() ArgParser { return new(CheckKarmaArgs) },
		"top":      func() ArgParser { return new(LeaderboardArgs) },
		"bot":      func() ArgParser { return new(LoserboardArgs) },
	}

	// install handlers
	// the prefix requires the message to be prefaced with the bot's name
	prefix := `^(` + r.name + `)`
	for cmd, ctor := range handlers {
		r.handlers[regexp.MustCompile(prefix+"\\s+"+cmd)] = ctor
	}

	return &r
}

func (r *Router) Route(s string) (args ArgParser, remainder string) {
	for matcher, action := range r.handlers {
		if matched := matcher.ReplaceAllString(s, ""); matched != s {
			return action(), matched
		}
	}

	return new(ChangeKarmaArgs), s
}
