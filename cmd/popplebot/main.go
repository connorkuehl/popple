package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sethvargo/go-retry"

	"github.com/connorkuehl/popple"
	"github.com/connorkuehl/popple/cmd/popplebot/internal/discord"
	"github.com/connorkuehl/popple/cmd/popplebot/internal/rabbitmq"
	"github.com/connorkuehl/popple/cmd/popplebot/internal/service"
	"github.com/connorkuehl/popple/internal/event"
)

var (
	token = os.Getenv("POPPLEBOT_DISCORD_TOKEN")

	amqpHost = os.Getenv("POPPLEBOT_AMQP_HOST")
	amqpPort = os.Getenv("POPPLEBOT_AMQP_PORT")
	amqpUser = os.Getenv("POPPLEBOT_AMQP_USER")
	amqpPass = os.Getenv("POPPLEBOT_AMQP_PASS")
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := run(ctx); err != nil {
		log.Fatalln(err)
	}
}

func run(ctx context.Context) error {
	runManifest := map[string]interface{}{
		"rabbitmq-host": amqpHost,
		"rabbitmq-port": amqpPort,
		"rabbitmq-user": amqpUser,
	}

	structured, _ := json.Marshal(runManifest)
	log.Println(string(structured))

	connctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var conn *amqp.Connection
	err := retry.Fibonacci(connctx, 1*time.Second, func(ctx context.Context) error {
		var err error
		conn, err = amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s", amqpUser, amqpPass, amqpHost, amqpPort))
		if err != nil {
			log.Println("failed to connect to event bus,", err, "retrying")
			return retry.RetryableError(err)
		}
		return nil
	})
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

	messageStream := discord.MessageStream(ctx, session)
	eventStream := event.Stream(ctx, events)

	var svc service.Service = service.New(
		rabbitmq.NewRequestPublisher(ch, requestQueue.Name),
		discord.New(session),
		popple.NewMux("@"+session.State.User.Username),
	)

	var wg sync.WaitGroup
	defer wg.Wait()

	wg.Add(2)
	go requestLoop(ctx, &wg, svc, messageStream)
	go eventLoop(ctx, &wg, svc, eventStream)

	return nil
}

func requestLoop(
	ctx context.Context,
	wg *sync.WaitGroup,
	svc service.Service,
	messages <-chan discord.Message,
) {
	defer wg.Done()
	defer log.Println("request loop has exited")
	log.Println("request loop is ready")

	for {
		select {
		case <-ctx.Done():
			log.Println("request loop exiting:", ctx.Err())
			return
		case message, ok := <-messages:
			if !ok {
				log.Println("request loop exiting: request chan is closed")
				return
			}
			err := svc.HandleDiscordMessage(ctx, message)
			if err != nil {
				log.Println("failed to handle Discord message:", err)
			}
		}
	}
}

func eventLoop(
	ctx context.Context,
	wg *sync.WaitGroup,
	svc service.Service,
	events <-chan event.Event,
) {
	defer wg.Done()
	defer log.Println("event loop has exited")
	log.Println("event loop is ready")

	for {
		select {
		case <-ctx.Done():
			log.Println("event loop exiting:", ctx.Err())
			return
		case event, ok := <-events:
			if !ok {
				log.Println("event loop exiting: event chan is closed")
				return
			}

			structured, _ := json.Marshal(event)
			log.Println(string(structured))

			var err error
			switch {
			case event.ChangedAnnounce != nil:
				err = svc.HandleChangedAnnounce(ctx, event.ChangedAnnounce)
			case event.ChangedKarma != nil:
				err = svc.HandleChangedKarma(ctx, event.ChangedKarma)
			case event.CheckedKarma != nil:
				err = svc.HandleCheckedKarma(ctx, event.CheckedKarma)
			case event.CheckedLeaderboard != nil:
				err = svc.HandleCheckedLeaderboard(ctx, event.CheckedLeaderboard)
			case event.CheckedLoserboard != nil:
				err = svc.HandleCheckedLoserboard(ctx, event.CheckedLoserboard)
			}
			if err != nil {
				log.Println("event handling failed:", err)
			}
		}
	}
}
