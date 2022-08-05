package discordtest

type Reaction struct {
	ChannelID string
	MessageID string
	EmojiID   string
}

type Message struct {
	ChannelID string
	Contents  string
}

type Response struct {
	Reaction *Reaction
	Message  *Message
}

type ResponseRecorder struct {
	Responses []Response
}

func NewResponseRecorder() *ResponseRecorder {
	return new(ResponseRecorder)
}

func (r *ResponseRecorder) ReactToMessage(channelID, messageID, emoji string) error {
	rsp := Response{
		Reaction: &Reaction{
			ChannelID: channelID,
			MessageID: messageID,
			EmojiID:   emoji,
		},
	}
	r.Responses = append(r.Responses, rsp)
	return nil
}

func (r *ResponseRecorder) SendMessage(channelID, message string) error {
	rsp := Response{
		Message: &Message{
			ChannelID: channelID,
			Contents:  message,
		},
	}
	r.Responses = append(r.Responses, rsp)
	return nil
}
