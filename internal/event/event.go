package event

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Event struct {
	RequestChangeAnnounce   *RequestChangeAnnounce   `json:"request-change-announce,omitempty"`
	RequestBumpKarma        *RequestBumpKarma        `json:"request-bump-karma,omitempty"`
	RequestCheckKarma       *RequestCheckKarma       `json:"request-check-karma,omitempty"`
	RequestCheckLeaderboard *RequestCheckLeaderboard `json:"request-check-leaderboard,omitempty"`
	RequestCheckLoserboard  *RequestCheckLoserboard  `json:"request-check-loserboard,omitempty"`
	CheckedKarma            *CheckedKarma            `json:"event-checked-karma,omitempty"`
	CheckedLeaderboard      *CheckedLeaderboard      `json:"event-checked-leaderboard,omitempty"`
	CheckedLoserboard       *CheckedLoserboard       `json:"event-checked-loserboard,omitempty"`
	ChangedKarma            *ChangedKarma            `json:"event-changed-karma,omitempty"`
	ChangedAnnounce         *ChangedAnnounce         `json:"event-changed-announce,omitempty"`
}

type ReplyTo struct {
	ChannelID string `json:"channel_id"`
}

type ReactTo struct {
	ChannelID string `json:"channel_id"`
	MessageID string `json:"message_id"`
}

type RequestChangeAnnounce struct {
	ReactTo    ReactTo `json:"react_to"`
	ServerID   string  `json:"server_id"`
	NoAnnounce bool    `json:"no_announce"`
}

type RequestBumpKarma struct {
	ReplyTo  ReplyTo          `json:"reply_to"`
	ServerID string           `json:"server_id"`
	Who      map[string]int64 `json:"who"`
}

type RequestCheckKarma struct {
	ReplyTo  ReplyTo             `json:"reply_to"`
	ServerID string              `json:"server_id"`
	Who      map[string]struct{} `json:"who"`
}

type RequestCheckLeaderboard struct {
	ReplyTo  ReplyTo `json:"reply_to"`
	ServerID string  `json:"server_id"`
	Limit    uint    `json:"limit"`
}

type RequestCheckLoserboard struct {
	ReplyTo  ReplyTo `json:"reply_to"`
	ServerID string  `json:"server_id"`
	Limit    uint    `json:"limit"`
}

type CheckedKarma struct {
	ReplyTo ReplyTo          `json:"reply_to"`
	Who     map[string]int64 `json:"who"`
}

type Score struct {
	Name  string `json:"name"`
	Karma int64  `json:"karma"`
}

type CheckedLeaderboard struct {
	ReplyTo ReplyTo `json:"reply_to"`
	Board   []Score `json:"leaderboard"`
}

type CheckedLoserboard struct {
	ReplyTo ReplyTo `json:"reply_to"`
	Board   []Score `json:"loserboard"`
}

type ChangedKarma struct {
	ReplyTo  ReplyTo          `json:"reply_to"`
	Who      map[string]int64 `json:"who"`
	Announce bool             `json:"announce"`
}

type ChangedAnnounce struct {
	ReactTo ReactTo `json:"react_to"`
}

func Stream(ctx context.Context, deliveries <-chan amqp.Delivery) <-chan Event {
	ch := make(chan Event)
	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case delivery, ok := <-deliveries:
				if !ok {
					return
				}

				var evt Event
				err := json.Unmarshal(delivery.Body, &evt)
				if err != nil {
					log.Println("failed to deserialize event:", err)
					continue
				}

				ch <- evt
			}
		}
	}()
	return ch
}
