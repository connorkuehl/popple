package service

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/alecthomas/repr"

	"github.com/connorkuehl/popple"
	"github.com/connorkuehl/popple/cmd/popplebot/internal/discord"
	"github.com/connorkuehl/popple/cmd/popplebot/internal/discord/discordtest"
	"github.com/connorkuehl/popple/cmd/popplebot/internal/rabbitmq/rabbitmqtest"
	"github.com/connorkuehl/popple/internal/event"
)

func TestHandleCheckedKarma(t *testing.T) {
	t.Run("it emits karma counts for requested entities", func(t *testing.T) {
		disc := discordtest.NewResponseRecorder()
		svc := New(
			nil,
			disc,
			nil,
		)

		input := event.CheckedKarma{
			ReplyTo: event.ReplyTo{
				ChannelID: "2324",
			},
			Who: map[string]int64{
				"poe the potato pirate":       20,
				"oakley the golden retriever": 9001,
			},
		}

		err := svc.HandleCheckedKarma(context.Background(), &input)
		if err != nil {
			t.Error(err)
		}

		if len(disc.Responses) != 1 {
			t.Errorf("expected 1 response, got %s", repr.String(disc.Responses))
		}

		rsp := disc.Responses[0].Message
		if rsp.ChannelID != "2324" {
			t.Errorf("expected ChannelID %q, got %q", "2324", rsp.ChannelID)
		}

		want := map[string]int64{
			"poe the potato pirate":       20,
			"oakley the golden retriever": 9001,
		}
		for who, amt := range want {
			tuple := map[string]int64{who: amt}
			if !hasKarmaCounts(rsp.Contents, tuple) {
				t.Errorf("expected to find karma details for %s in message %q", repr.String(tuple), rsp.Contents)
			}
		}
	})
}

func TestHandleChangedAnnounce(t *testing.T) {
	t.Run("it reacts with a checkmark when it changes announce settings", func(t *testing.T) {
		disc := discordtest.NewResponseRecorder()
		svc := New(
			nil,
			disc,
			nil,
		)

		input := event.ChangedAnnounce{
			ReactTo: event.ReactTo{
				ChannelID: "5678",
				MessageID: "2011",
			},
		}

		err := svc.HandleChangedAnnounce(context.Background(), &input)
		if err != nil {
			t.Error(err)
		}

		want := []discordtest.Response{
			{Reaction: &discordtest.Reaction{MessageID: "2011", ChannelID: "5678", EmojiID: "✅"}},
		}

		if !reflect.DeepEqual(want, disc.Responses) {
			t.Errorf("want responses %s, got responses %s", repr.String(want), repr.String(disc.Responses))
		}
	})
}

func TestHandleChangedKarma(t *testing.T) {
	t.Run("it doesn't announce if it is configured not to", func(t *testing.T) {
		disc := discordtest.NewResponseRecorder()
		svc := New(
			nil,
			disc,
			nil,
		)

		input := event.ChangedKarma{
			ReplyTo: event.ReplyTo{
				ChannelID: "4321",
			},
			Who:      map[string]int64{"hjkl": 1},
			Announce: false,
		}

		err := svc.HandleChangedKarma(context.Background(), &input)
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

func TestHandleDiscordMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    discord.Message
		wantDisc []discordtest.Response
		wantEvnt []*event.Event
		wantErr  error
	}{
		{
			name: "announce emits err if missing arg",
			input: discord.Message{
				ID:        "1",
				ChannelID: "2",
				ServerID:  "3",
				Body:      "bot announce",
			},
			wantDisc: []discordtest.Response{
				{Reaction: &discordtest.Reaction{
					MessageID: "1",
					ChannelID: "2",
					EmojiID:   "❓",
				}},
				{Message: &discordtest.Message{
					ChannelID: "2",
					Contents:  `Valid announce settings are: "on", "off", "yes", "no"`,
				}},
			},
		},
		{
			name: "announce emits err if arg invalid",
			input: discord.Message{
				ID:        "2",
				ChannelID: "3",
				ServerID:  "4",
				Body:      "bot announce hjkl",
			},
			wantDisc: []discordtest.Response{
				{Reaction: &discordtest.Reaction{
					MessageID: "2",
					ChannelID: "3",
					EmojiID:   "❓",
				}},
				{Message: &discordtest.Message{
					ChannelID: "3",
					Contents:  `Valid announce settings are: "on", "off", "yes", "no"`,
				}},
			},
		},
		{
			name: "announce submits request if args ok",
			input: discord.Message{
				ID:        "3",
				ChannelID: "4",
				ServerID:  "5",
				Body:      "bot announce on",
			},
			wantEvnt: []*event.Event{
				{
					RequestChangeAnnounce: &event.RequestChangeAnnounce{
						ReactTo: event.ReactTo{
							ChannelID: "4",
							MessageID: "3",
						},
						ServerID:   "5",
						NoAnnounce: false,
					},
				},
			},
		},
		{
			name: "bump karma does not submit request if no entities",
			input: discord.Message{
				ID:        "4",
				ChannelID: "5",
				ServerID:  "6",
				Body:      "hello world",
			},
		},
		{
			name: "bump karma submits request if there are entities",
			input: discord.Message{
				ID:        "5",
				ChannelID: "6",
				ServerID:  "7",
				Body:      "hello++ world",
			},
			wantEvnt: []*event.Event{{
				RequestBumpKarma: &event.RequestBumpKarma{
					ReplyTo: event.ReplyTo{
						ChannelID: "6",
					},
					ServerID: "7",
					Who:      map[string]int64{"hello": 1},
				},
			}},
		},
		{
			name: "karma reacts if no entities are listed",
			input: discord.Message{
				ID:        "6",
				ChannelID: "7",
				ServerID:  "8",
				Body:      "bot karma",
			},
			wantDisc: []discordtest.Response{{
				Reaction: &discordtest.Reaction{
					ChannelID: "7",
					MessageID: "6",
					EmojiID:   "❓",
				},
			}},
		},
		{
			name: "karma submits request if arguments",
			input: discord.Message{
				ID:        "7",
				ChannelID: "8",
				ServerID:  "9",
				Body:      "bot karma potato",
			},
			wantEvnt: []*event.Event{{
				RequestCheckKarma: &event.RequestCheckKarma{
					ReplyTo: event.ReplyTo{
						ChannelID: "8",
					},
					ServerID: "9",
					Who:      map[string]struct{}{"potato": {}},
				},
			}},
		},
		{
			name: "leaderboard replies with err if arg invalid",
			input: discord.Message{
				ID:        "8",
				ChannelID: "9",
				ServerID:  "10",
				Body:      "bot top 0",
			},
			wantDisc: []discordtest.Response{{
				Message: &discordtest.Message{
					ChannelID: "9",
					Contents:  "The number of entries to list must be a positive non-zero integer",
				},
			}},
		},
		{
			name: "leaderboard submits request if arg ok",
			input: discord.Message{
				ID:        "9",
				ChannelID: "10",
				ServerID:  "11",
				Body:      "bot top 3",
			},
			wantEvnt: []*event.Event{{
				RequestCheckLeaderboard: &event.RequestCheckLeaderboard{
					ReplyTo: event.ReplyTo{
						ChannelID: "10",
					},
					ServerID: "11",
					Limit:    3,
				},
			}},
		},
		{
			name: "loserboard replies with err if arg invalid",
			input: discord.Message{
				ID:        "10",
				ChannelID: "11",
				ServerID:  "12",
				Body:      "bot bot 0",
			},
			wantDisc: []discordtest.Response{{
				Message: &discordtest.Message{
					ChannelID: "11",
					Contents:  "The number of entries to list must be a positive non-zero integer",
				},
			}},
		},
		{
			name: "loserboard submits request if arg ok",
			input: discord.Message{
				ID:        "11",
				ChannelID: "12",
				ServerID:  "13",
				Body:      "bot bot 3",
			},
			wantEvnt: []*event.Event{{
				RequestCheckLoserboard: &event.RequestCheckLoserboard{
					ReplyTo: event.ReplyTo{
						ChannelID: "12",
					},
					ServerID: "13",
					Limit:    3,
				},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventbus := rabbitmqtest.NewEventRecorder()
			mux := popple.NewMux("bot")
			disc := discordtest.NewResponseRecorder()
			svc := New(eventbus, disc, mux)

			err := svc.HandleDiscordMessage(context.Background(), tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("wanted err=%v, got=%v", tt.wantErr, err)
			}

			if !reflect.DeepEqual(tt.wantDisc, disc.Responses) {
				t.Errorf("wanted discord responses %s, got responses %s", repr.String(tt.wantDisc), repr.String(disc.Responses))
			}

			if !reflect.DeepEqual(tt.wantEvnt, eventbus.Events) {
				t.Errorf("wanted events %s, got events %s", repr.String(tt.wantEvnt), repr.String(eventbus.Events))
			}
		})
	}
}

func hasKarmaCounts(s string, who map[string]int64) bool {
	for who, amt := range who {
		if !strings.Contains(s, fmt.Sprintf("%s has %d karma", who, amt)) {
			return false
		}
	}
	return true
}
