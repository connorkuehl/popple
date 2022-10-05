package bot

import (
	"testing"

	"github.com/jaswdr/faker"
	"github.com/stretchr/testify/mock"

	"github.com/connorkuehl/popple"
	"github.com/connorkuehl/popple/cmd/popplebot/internal/command"
	"github.com/connorkuehl/popple/cmd/popplebot/internal/discord"
)

func TestListen(t *testing.T) {
	t.Run("it emits valid values for set announce if input is bad", func(t *testing.T) {
		tests := []struct {
			input string
		}{
			{input: "popple announce hjkl"},
			{input: "popple announce"},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				faker := faker.New()
				mockDiscord := NewMockDiscord(t)
				mockPopple := NewMockPoppleClient(t)
				router := command.NewRouter("popple")
				bot := New(mockPopple, mockDiscord, router)

				messages := make(chan discord.Message, 1)
				message := discord.Message{
					ID:        faker.UUID().V4(),
					GuildID:   faker.UUID().V4(),
					ChannelID: faker.UUID().V4(),
					Content:   tt.input,
				}
				messages <- message
				close(messages)

				mockDiscord.EXPECT().
					SendMessageToChannel(message.ChannelID, `Valid announce settings are "yes", "on", "no", "off"`).
					Return(nil).
					Once()

				err := bot.Listen(messages)
				if err != nil {
					t.Error(err)
				}
			})
		}
	})

	t.Run("it reacts with a checkmark emoji when set announce succeeds", func(t *testing.T) {
		tests := []struct {
			input      string
			noAnnounce bool
		}{
			{input: "popple announce yes", noAnnounce: false},
			{input: "popple announce on", noAnnounce: false},
			{input: "popple announce off", noAnnounce: true},
			{input: "popple announce no", noAnnounce: true},
		}

		for _, tt := range tests {
			faker := faker.New()
			mockDiscord := NewMockDiscord(t)
			mockPopple := NewMockPoppleClient(t)
			router := command.NewRouter("popple")
			bot := New(mockPopple, mockDiscord, router)

			messages := make(chan discord.Message, 1)
			message := discord.Message{
				ID:        faker.UUID().V4(),
				GuildID:   faker.UUID().V4(),
				ChannelID: faker.UUID().V4(),
				Content:   tt.input,
			}
			messages <- message
			close(messages)

			mockDiscord.EXPECT().
				ReactToMessageWithEmoji(message.ChannelID, message.ID, "✅").
				Return(nil).
				Once()

			mockPopple.EXPECT().
				PutConfig(mock.Anything, &popple.Config{ServerID: message.GuildID, NoAnnounce: tt.noAnnounce}).
				Return(nil).
				Once()

			err := bot.Listen(messages)
			if err != nil {
				t.Error(err)
			}
		}
	})
}
