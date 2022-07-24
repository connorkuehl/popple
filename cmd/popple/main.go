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

	"github.com/connorkuehl/popple"
	"github.com/connorkuehl/popple/cmd/popple/internal/service"
	poperrs "github.com/connorkuehl/popple/errors"
	"github.com/connorkuehl/popple/event"
	mysqlrepo "github.com/connorkuehl/popple/repo/mysql"
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

	repo := mysqlrepo.New(db)

	var svc service.Service = service.New(repo)
	svc = service.NewLogged(svc)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		details, ok := svc.Health()
		details["healthy"] = ok

		if err := json.NewEncoder(w).Encode(details); err != nil {
			log.Printf("failed to encode health checks: %v", err)
		}
	})
	go func() {
		log.Println("health checks:", listenHealth+"/health")
		log.Println(http.ListenAndServe(listenHealth, nil))
	}()

	log.Println("ready to dole out some karma")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case req, ok := <-requests:
			if !ok {
				return errors.New("consumer has seen a closed requests channel")
			}

			var evt event.Event
			err := json.Unmarshal(req.Body, &evt)
			if err != nil {
				log.Println("failed to deserialize event:", err)
				continue
			}

			eventJSON, _ := json.Marshal(evt)
			log.Println("got event", string(eventJSON))

			switch {
			case evt.RequestChangeAnnounce != nil:
				req := evt.RequestChangeAnnounce
				err := popple.Announce(repo, req.ServerID, !req.NoAnnounce)
				if err != nil {
					log.Println("request failed", req, "error:", err)
					continue
				}

				payload, err := json.Marshal(event.Event{
					ChangedAnnounce: &event.ChangedAnnounce{
						ReactTo: req.ReactTo,
					},
				})
				if err != nil {
					log.Println("failed to encode:", err)
					continue
				}
				err = ch.PublishWithContext(
					context.TODO(),
					"popple_topic",
					"changed.announce",
					false,
					false,
					amqp.Publishing{
						Body: payload,
					},
				)
				if err != nil {
					log.Println("failed to publish:", err)
					continue
				}
			case evt.RequestBumpKarma != nil:
				req := evt.RequestBumpKarma
				incr, err := popple.BumpKarma(repo, req.ServerID, req.Who)
				if err != nil {
					log.Println("request failed", req, "error:", err)
					continue
				}

				cfg, err := repo.Config(req.ServerID)
				if errors.Is(err, poperrs.ErrNotFound) {
					err = nil
				}
				if err != nil {
					log.Println("failed to check config:", err)
					continue
				}

				payload, err := json.Marshal(event.Event{
					ChangedKarma: &event.ChangedKarma{
						ReplyTo:  req.ReplyTo,
						Who:      incr,
						Announce: !cfg.NoAnnounce,
					},
				})
				if err != nil {
					log.Println("failed to encode:", err)
					continue
				}
				err = ch.PublishWithContext(
					context.TODO(),
					"popple_topic",
					"changed.karma",
					false,
					false,
					amqp.Publishing{
						Body: payload,
					},
				)
				if err != nil {
					log.Println("failed to publish:", err)
					continue
				}
			case evt.RequestCheckKarma != nil:
				req := evt.RequestCheckKarma
				who, err := popple.Karma(repo, req.ServerID, req.Who)
				if err != nil {
					log.Println("request failed", req, "error:", err)
					continue
				}

				payload, err := json.Marshal(event.Event{
					CheckedKarma: &event.CheckedKarma{
						ReplyTo: req.ReplyTo,
						Who:     who,
					},
				})
				if err != nil {
					log.Println("failed to encode:", err)
					continue
				}
				err = ch.PublishWithContext(
					context.TODO(),
					"popple_topic",
					"checked.karma",
					false,
					false,
					amqp.Publishing{
						Body: payload,
					},
				)
				if err != nil {
					log.Println("failed to publish:", err)
					continue
				}
			case evt.RequestCheckLeaderboard != nil:
				req := evt.RequestCheckLeaderboard
				top, err := popple.Leaderboard(repo, req.ServerID, req.Limit)
				if err != nil {
					log.Println("request failed", req, "error:", err)
					continue
				}

				board := make([]event.Score, 0, len(top))
				for _, s := range top {
					board = append(board, event.Score{Name: s.Name, Karma: s.Karma})
				}

				payload, err := json.Marshal(event.Event{
					CheckedLeaderboard: &event.CheckedLeaderboard{
						ReplyTo: req.ReplyTo,
						Board:   board,
					},
				})
				if err != nil {
					log.Println("failed to encode:", err)
					continue
				}
				err = ch.PublishWithContext(
					context.TODO(),
					"popple_topic",
					"checked.karma",
					false,
					false,
					amqp.Publishing{
						Body: payload,
					},
				)
				if err != nil {
					log.Println("failed to publish:", err)
					continue
				}
			case evt.RequestCheckLoserboard != nil:
				req := evt.RequestCheckLoserboard
				top, err := popple.Loserboard(repo, req.ServerID, req.Limit)
				if err != nil {
					log.Println("request failed", req, "error:", err)
					continue
				}

				board := make([]event.Score, 0, len(top))
				for _, s := range top {
					board = append(board, event.Score{Name: s.Name, Karma: s.Karma})
				}

				payload, err := json.Marshal(event.Event{
					CheckedLoserboard: &event.CheckedLoserboard{
						ReplyTo: req.ReplyTo,
						Board:   board,
					},
				})
				if err != nil {
					log.Println("failed to encode:", err)
					continue
				}
				err = ch.PublishWithContext(
					context.TODO(),
					"popple_topic",
					"checked.karma",
					false,
					false,
					amqp.Publishing{
						Body: payload,
					},
				)
				if err != nil {
					log.Println("failed to publish:", err)
					continue
				}
			default:
				log.Println("received unknown request", evt)
			}
		}
	}
}
