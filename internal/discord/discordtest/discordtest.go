package discordtest

import "github.com/connorkuehl/popple/internal/discord"

type Reaction struct {
	ChannelID string
	MessageID string
	Emoji     string
}

type Message struct {
	ChannelID string
	Content   string
}

type Response struct {
	Reaction Reaction
	Message  Message
}

type ResponseRecorder struct {
	Responses []Response
	messages  []discord.Message
}

func NewResponseRecorder(messages []discord.Message) *ResponseRecorder {
	return &ResponseRecorder{messages: messages}
}

func (r *ResponseRecorder) SendMessageToChannel(channelID string, msg string) error {
	r.Responses = append(r.Responses, Response{Message: Message{ChannelID: channelID, Content: msg}})
	return nil
}

func (r *ResponseRecorder) ReactToMessageWithEmoji(channelID, messageID, emojiID string) error {
	r.Responses = append(r.Responses, Response{Reaction: Reaction{ChannelID: channelID, MessageID: messageID, Emoji: emojiID}})
	return nil
}

func (r *ResponseRecorder) Messages() <-chan discord.Message {
	ch := make(chan discord.Message, len(r.messages))
	for _, msg := range r.messages {
		ch <- msg
	}
	close(ch)
	return ch
}
