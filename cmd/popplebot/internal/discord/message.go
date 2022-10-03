package discord

import (
	"context"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type Message struct {
	ID        string
	GuildID   string
	ChannelID string
	Content   string
}

func Messages(ctx context.Context, session *discordgo.Session) <-chan Message {
	ch := make(chan Message, 16)

	var closeOnce sync.Once
	session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
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

		select {
		case <-ctx.Done():
			closeOnce.Do(func() { close(ch) })
			return
		case ch <- msg:
		}
	})

	go func() {
		<-ctx.Done()
		closeOnce.Do(func() { close(ch) })
	}()

	return ch
}
