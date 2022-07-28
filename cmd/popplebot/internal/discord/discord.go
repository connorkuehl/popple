package discord

import (
	"context"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type Message struct {
	ID        string
	ChannelID string
	ServerID  string
	Body      string
}

type Session interface {
	AddHandler(interface{}) func()
}

func MessageStream(ctx context.Context, session Session) <-chan Message {
	ch := make(chan Message)

	var closeStreamOnce sync.Once
	handler := func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if s.State.User.ID == m.Author.ID {
			return
		}

		if len(m.GuildID) == 0 {
			return
		}

		message := Message{
			ID:        m.ID,
			ChannelID: m.ChannelID,
			ServerID:  m.GuildID,
			Body:      strings.TrimSpace(m.ContentWithMentionsReplaced()),
		}

		select {
		case <-ctx.Done():
			closeStreamOnce.Do(func() { close(ch) })
			return
		case ch <- message:
		}
	}

	detach := session.AddHandler(handler)
	go func() {
		<-ctx.Done()
		detach()
	}()

	return ch
}

type Discord struct {
	*discordgo.Session
}

func New(session *discordgo.Session) *Discord {
	return &Discord{session}
}

func (d *Discord) ReactToMessage(channelID, messageID, emoji string) error {
	return d.MessageReactionAdd(channelID, messageID, emoji)
}

func (d *Discord) SendMessage(channelID, message string) error {
	_, err := d.ChannelMessageSend(channelID, message)
	return err
}
