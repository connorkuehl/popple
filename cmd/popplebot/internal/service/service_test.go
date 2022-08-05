package service

import (
	"context"
	"testing"

	"github.com/alecthomas/repr"

	"github.com/connorkuehl/popple/cmd/popplebot/internal/discord/discordtest"
	"github.com/connorkuehl/popple/internal/event"
)

func TestHandleLeaderboard(t *testing.T) {
	t.Run("it doesn't reply if the leaderboard is empty", func(t *testing.T) {
		t.Skip("https://github.com/connorkuehl/popple/issues/118")

		disc := discordtest.NewResponseRecorder()
		svc := New(
			nil,  // service should not interact with eventbus for this test
			disc, // discord
			nil,  // service won't need mux for this test
		)

		input := event.CheckedLeaderboard{
			ReplyTo: event.ReplyTo{
				ChannelID: "1234",
			},
			Board: []event.Score{},
		}

		err := svc.HandleCheckedLeaderboard(context.Background(), &input)
		if err != nil {
			t.Error(err)
		}

		got := len(disc.Responses)
		want := 0
		if got != want {
			t.Errorf("want %d responses, got %s", want, repr.String(disc.Responses))
		}
	})
}

func TestHandleLoserboard(t *testing.T) {
	t.Run("it doesn't reply if the loserboard is empty", func(t *testing.T) {
		t.Skip("https://github.com/connorkuehl/popple/issues/118")

		disc := discordtest.NewResponseRecorder()
		svc := New(
			nil,  // service should not interact with eventbus for this test
			disc, // discord
			nil,  // service won't need mux for this test
		)

		input := event.CheckedLoserboard{
			ReplyTo: event.ReplyTo{
				ChannelID: "2345",
			},
			Board: []event.Score{},
		}

		err := svc.HandleCheckedLoserboard(context.Background(), &input)
		if err != nil {
			t.Error(err)
		}

		got := len(disc.Responses)
		want := 0
		if got != want {
			t.Errorf("want %d responses, got %s", want, repr.String(disc.Responses))
		}
	})
}
