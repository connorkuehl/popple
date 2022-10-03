package discord

import "github.com/bwmarrin/discordgo"

type Session struct {
	s *discordgo.Session
}

func NewSession(s *discordgo.Session) *Session {
	return &Session{
		s: s,
	}
}

func (s *Session) SendMessageToChannel(channelID string, msg string) error {
	_, err := s.s.ChannelMessageSend(channelID, msg)
	return err
}

func (s *Session) ReactToMessageWithEmoji(channelID, messageID, emojiID string) error {
	return s.s.MessageReactionAdd(channelID, messageID, emojiID)
}
