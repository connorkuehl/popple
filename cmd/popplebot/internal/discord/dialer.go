package discord

import "github.com/bwmarrin/discordgo"

type Dialer struct {
	token string
}

func NewDialer(token string) *Dialer {
	return &Dialer{
		token: token,
	}
}

func (d *Dialer) Dial() (*discordgo.Session, error) {
	session, err := discordgo.New("Bot " + d.token)
	if err != nil {
		return nil, err
	}

	session.Identify.Intents |= discordgo.IntentMessageContent
	err = session.Open()
	if err != nil {
		return nil, err
	}

	return session, nil
}
