package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/go-sql-driver/mysql"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/connorkuehl/popple/cmd/popplesvc/internal/rabbitmq"
	mysqlrepo "github.com/connorkuehl/popple/cmd/popplesvc/internal/repo/mysql"
	"github.com/connorkuehl/popple/cmd/popplesvc/internal/service"
	"github.com/connorkuehl/popple/internal/event"
)

var (
	listenHealth = os.Getenv("POPPLE_LISTEN_HEALTH")

	dbHost = os.Getenv("POPPLE_DB_HOST")
	dbPort = os.Getenv("POPPLE_DB_PORT")
	dbUser = os.Getenv("POPPLE_DB_USER")
	dbPass = os.Getenv("POPPLE_DB_PASS")
	dbName = os.Getenv("POPPLE_DB_NAME")

	amqpHost = os.Getenv("POPPLE_AMQP_HOST")
	amqpPort = os.Getenv("POPPLE_AMQP_PORT")
	amqpUser = os.Getenv("POPPLE_AMQP_USER")
	amqpPass = os.Getenv("POPPLE_AMQP_PASS")
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := run(ctx); err != nil {
		log.Fatalln(err)
	}
}

func run(ctx context.Context) error {
	dbcfg := mysql.Config{
		User:      dbUser,
		Passwd:    dbPass,
		Net:       "tcp",
		Addr:      fmt.Sprintf("%s:%s", dbHost, dbPort),
		DBName:    dbName,
		ParseTime: true,
	}
	db, err := sql.Open("mysql", dbcfg.FormatDSN())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

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

	requests, err := ch.Consume(
		requestQueue.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	var svc service.Service = service.New(
		rabbitmq.NewEventBus(ch),
		mysqlrepo.New(db),
	)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		details := map[string]interface{}{
			"healthy": true,
		}

		if err := json.NewEncoder(w).Encode(details); err != nil {
			log.Printf("failed to encode health checks: %v", err)
		}
	})
	go func() {
		log.Println("health checks:", listenHealth+"/health")
		log.Println(http.ListenAndServe(listenHealth, nil))
	}()

	log.Println("ready to dole out some karma")

	eventLoop(ctx, svc, event.Stream(ctx, requests))
	return nil
}

func eventLoop(ctx context.Context, svc service.Service, events <-chan event.Event) {
	defer log.Println("event loop has exited")
	log.Println("event loop is ready")

	for {
		select {
		case <-ctx.Done():
			log.Println("event loop exiting:", ctx.Err())
			return
		case evt, ok := <-events:
			if !ok {
				log.Println("event loop exiting: event chan is closed")
				return
			}

			structured, _ := json.Marshal(evt)
			log.Println(string(structured))

			var err error
			switch {
			case evt.RequestCheckKarma != nil:
				err = svc.CheckKarma(ctx, evt.RequestCheckKarma)
			case evt.RequestCheckLeaderboard != nil:
				err = svc.CheckLeaderboard(ctx, evt.RequestCheckLeaderboard)
			case evt.RequestCheckLoserboard != nil:
				err = svc.CheckLoserboard(ctx, evt.RequestCheckLoserboard)
			case evt.RequestChangeAnnounce != nil:
				err = svc.ChangeAnnounce(ctx, evt.RequestChangeAnnounce)
			case evt.RequestBumpKarma != nil:
				err = svc.ChangeKarma(ctx, evt.RequestBumpKarma)
			default:
				err = errors.New("unknown event type")
			}
			if err != nil {
				log.Println("event handling failed:", err)
			}
		}
	}
}
