package discord

import (
	"os"

	"github.com/bwmarrin/discordgo"
)

type Token string

func TokenFromEnv() (Token, error) {
	return tokenFromEnv(os.Getenv)
}

type Dialer struct {
	token Token
}

func NewDialer(token Token) *Dialer {
	return &Dialer{token: token}
}

func (d *Dialer) Dial() (*Session, error) {
	session, err := discordgo.New("Bot " + string(d.token))
	if err != nil {
		return nil, err
	}

	session.Identify.Intents |= discordgo.IntentMessageContent
	err = session.Open()
	if err != nil {
		return nil, err
	}

	return &Session{s: session}, err
}

type Session struct {
	s        *discordgo.Session
	messages chan Message
}

func NewSession(dialer *Dialer) (*Session, func(), error) {
	s, err := dialer.Dial()
	if err != nil {
		return nil, nil, err
	}

	s.messages = make(chan Message)
	ch := s.messages

	detach := s.s.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore messages from self.
		if s.State.User.Username == m.Author.Username {
			return
		}

		// No DMs.
		if len(m.GuildID) == 0 {
			return
		}

		msg := Message{
			ID:        m.ID,
			GuildID:   m.GuildID,
			ChannelID: m.ChannelID,
			Content:   m.ContentWithMentionsReplaced(),
		}

		ch <- msg
	})

	return s, func() {
		detach()
		_ = s.s.Close()
	}, nil
}

func (s *Session) SendMessageToChannel(channelID string, msg string) error {
	_, err := s.s.ChannelMessageSend(channelID, msg)
	return err
}

func (s *Session) ReactToMessageWithEmoji(channelID, messageID, emojiID string) error {
	return s.s.MessageReactionAdd(channelID, messageID, emojiID)
}

func (s *Session) Username() string {
	return s.s.State.User.Username
}

func (s *Session) Messages() <-chan Message {
	return s.messages
}

type Message struct {
	ID        string
	GuildID   string
	ChannelID string
	Content   string
}
