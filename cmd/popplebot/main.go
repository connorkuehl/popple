package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"text/template"

	"github.com/bwmarrin/discordgo"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/connorkuehl/popple"
	poperrs "github.com/connorkuehl/popple/errors"
	"github.com/connorkuehl/popple/event"
)

var (
	token = os.Getenv("POPPLEBOT_DISCORD_TOKEN")

	amqpHost = os.Getenv("POPPLEBOT_AMQP_HOST")
	amqpPort = os.Getenv("POPPLEBOT_AMQP_PORT")
	amqpUser = os.Getenv("POPPLEBOT_AMQP_USER")
	amqpPass = os.Getenv("POPPLEBOT_AMQP_PASS")
)

var (
	templateLevels = template.Must(template.New("levels").Parse(`{{ range $name, $karma := . }}{{ $name }} has {{ $karma }} karma. {{ end }}`))
	templateBoard  = template.Must(template.New("board").Parse(
		`{{ range $entry := . }}* {{ $entry.Name }} has {{ $entry.Karma }} karma.
{{ end }}`))
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := run(ctx); err != nil {
		log.Fatalln(err)
	}
}

func run(ctx context.Context) error {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s", amqpUser, amqpPass, amqpHost, amqpPort))
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		"popple_topic",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	requestQueue, err := ch.QueueDeclare(
		"requests",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	queue, err := ch.QueueDeclare(
		"",
		false,
		false,
		true,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = ch.QueueBind(
		queue.Name,
		"checked.*",
		"popple_topic",
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = ch.QueueBind(
		queue.Name,
		"changed.*",
		"popple_topic",
		false,
		nil,
	)
	if err != nil {
		return err
	}

	events, err := ch.Consume(
		queue.Name,
		"popplebot",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return err
	}

	err = session.Open()
	if err != nil {
		return err
	}
	defer session.Close()
	log.Println("connected to Discord")

	var wg sync.WaitGroup
	wg.Add(2)
	go publisher(ctx, &wg, ch, requestQueue, session)
	go consumer(ctx, &wg, events, session)

	wg.Wait()
	return nil
}

func publisher(
	ctx context.Context,
	wg *sync.WaitGroup,
	ch *amqp.Channel,
	qu amqp.Queue,
	session *discordgo.Session,
) {
	defer wg.Done()
	defer log.Println("publisher has stopped")

	mux := popple.NewMux("@" + session.State.User.Username)
	detachMessageCreate := session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if s.State.User.ID == m.Author.ID {
			return
		}

		// Direct message
		if len(m.GuildID) == 0 {
			return
		}

		err := handleMessage(ctx, mux, ch, qu, s, m)
		if err != nil {
			log.Println("failed to handle message: %w", err)
			return
		}
	})
	defer detachMessageCreate()
	log.Println("publisher has started")

	<-ctx.Done()
}

func handleMessage(
	ctx context.Context,
	mux *popple.Mux,
	ch *amqp.Channel,
	qu amqp.Queue,
	session *discordgo.Session,
	msg *discordgo.MessageCreate,
) error {
	var req *event.Event

	message := strings.TrimSpace(msg.ContentWithMentionsReplaced())
	action, body := mux.Route(message)

	switch action.(type) {
	case popple.AnnounceHandler:
		on, err := popple.ParseAnnounceArgs(body)
		if errors.Is(err, poperrs.ErrMissingArgument) || errors.Is(err, poperrs.ErrInvalidArgument) {
			err := session.MessageReactionAdd(msg.ChannelID, msg.ID, "❓")
			if err != nil {
				return fmt.Errorf("failed to react to message: %w", err)
			}
			_, err = session.ChannelMessageSend(msg.ChannelID, `Valid announce settings are: "on", "off", "yes", "no"`)
			if err != nil {
				err = fmt.Errorf("failed to send message: %w", err)
			}
			return err
		}

		req = &event.Event{
			RequestChangeAnnounce: &event.RequestChangeAnnounce{
				ReactTo: event.ReactTo{
					ChannelID: msg.ChannelID,
					MessageID: msg.ID,
				},
				ServerID:   msg.GuildID,
				NoAnnounce: !on,
			}}
	case popple.BumpKarmaHandler:
		increments, _ := popple.ParseBumpKarmaArgs(body)
		for k, v := range increments {
			if v == 0 {
				delete(increments, k)
			}
		}
		if len(increments) == 0 {
			return nil
		}

		req = &event.Event{
			RequestBumpKarma: &event.RequestBumpKarma{
				ReplyTo: event.ReplyTo{
					ChannelID: msg.ChannelID,
				},
				ServerID: msg.GuildID,
				Who:      increments,
			}}
	case popple.KarmaHandler:
		who, err := popple.ParseKarmaArgs(body)
		if err != nil {
			err = session.MessageReactionAdd(msg.ChannelID, msg.ID, "❓")
			if err != nil {
				err = fmt.Errorf("message reaction add failed: %w", err)
			}
			return err
		}

		req = &event.Event{
			RequestCheckKarma: &event.RequestCheckKarma{
				ReplyTo: event.ReplyTo{
					ChannelID: msg.ChannelID,
				},
				ServerID: msg.GuildID,
				Who:      who,
			}}
	case popple.LeaderboardHandler:
		limit, err := popple.ParseLeaderboardArgs(body)
		if errors.Is(err, poperrs.ErrInvalidArgument) {
			_, err := session.ChannelMessageSend(msg.ChannelID, "The number of entries to list must be a positive non-zero integer")
			if err != nil {
				err = fmt.Errorf("message send failed: %w", err)
			}
			return err
		}

		req = &event.Event{
			RequestCheckLeaderboard: &event.RequestCheckLeaderboard{
				ReplyTo: event.ReplyTo{
					ChannelID: msg.ChannelID,
				},
				ServerID: msg.GuildID,
				Limit:    limit,
			}}
	case popple.LoserboardHandler:
		limit, err := popple.ParseLoserboardArgs(body)
		if errors.Is(err, poperrs.ErrInvalidArgument) {
			_, err := session.ChannelMessageSend(msg.ChannelID, "The number of entries to list must be a positive non-zero integer")
			if err != nil {
				err = fmt.Errorf("message send failed: %w", err)
			}
			return err
		}

		req = &event.Event{
			RequestCheckLoserboard: &event.RequestCheckLoserboard{
				ReplyTo: event.ReplyTo{
					ChannelID: msg.ChannelID,
				},
				ServerID: msg.GuildID,
				Limit:    limit,
			}}
	default:
		return fmt.Errorf("bug: reached default case for handle message: %q", message)
	}

	return publishRequest(ctx, ch, qu, req)
}

func publishRequest(ctx context.Context, ch *amqp.Channel, qu amqp.Queue, req *event.Event) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to serialize request: %w", err)
	}

	return ch.PublishWithContext(
		ctx,
		"",
		qu.Name,
		false,
		false,
		amqp.Publishing{
			Body: payload,
		},
	)
}

func consumer(
	ctx context.Context,
	wg *sync.WaitGroup,
	events <-chan amqp.Delivery,
	session *discordgo.Session,
) {
	defer wg.Done()
	defer log.Println("consumer has stopped")
	log.Println("consumer has started")

	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-events:
			if !ok {
				log.Println("consumer sees a closed events channel")
				return
			}

			var actual event.Event
			err := json.Unmarshal(evt.Body, &actual)
			if err != nil {
				log.Println("failed to deserialize event:", err)
				continue
			}

			eventJSON, _ := json.Marshal(actual)
			log.Println("got event", string(eventJSON))

			switch {
			case actual.CheckedKarma != nil:
				rsp := actual.CheckedKarma
				var r strings.Builder
				err := templateLevels.Execute(&r, rsp.Who)
				if err != nil {
					log.Println("failed to apply levels template:", err)
					continue
				}

				_, err = session.ChannelMessageSend(rsp.ReplyTo.ChannelID, r.String())
				if err != nil {
					log.Println("failed to send message:", err)
					continue
				}
			case actual.CheckedLeaderboard != nil:
				rsp := actual.CheckedLeaderboard
				var r strings.Builder
				err := templateBoard.Execute(&r, rsp.Board)
				if err != nil {
					log.Println("failed to apply board template:", err)
					continue
				}

				_, err = session.ChannelMessageSend(rsp.ReplyTo.ChannelID, r.String())
				if err != nil {
					log.Println("failed to send message:", err)
					continue
				}
			case actual.CheckedLoserboard != nil:
				rsp := actual.CheckedLoserboard
				var r strings.Builder
				err := templateBoard.Execute(&r, rsp.Board)
				if err != nil {
					log.Println("failed to apply board template:", err)
					continue
				}

				_, err = session.ChannelMessageSend(rsp.ReplyTo.ChannelID, r.String())
				if err != nil {
					log.Println("failed to send message:", err)
					continue
				}
			case actual.ChangedAnnounce != nil:
				rsp := actual.ChangedAnnounce
				err := session.MessageReactionAdd(rsp.ReactTo.ChannelID, rsp.ReactTo.MessageID, "✅")
				if err != nil {
					log.Println("failed to add reaction:", err)
					continue
				}
			case actual.ChangedKarma != nil:
				rsp := actual.ChangedKarma

				if !rsp.Announce {
					continue
				}

				var r strings.Builder
				err := templateLevels.Execute(&r, rsp.Who)
				if err != nil {
					log.Println("failed to apply levels template:", err)
					continue
				}

				_, err = session.ChannelMessageSend(rsp.ReplyTo.ChannelID, r.String())
				if err != nil {
					log.Println("failed to send message:", err)
					continue
				}
			default:
				log.Println("discarding unknown or unspecified event", evt)
			}
		}
	}
}
