package event

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/connorkuehl/popple/event"
)

func Stream(ctx context.Context, deliveries <-chan amqp.Delivery) <-chan event.Event {
	ch := make(chan event.Event)
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

				var evt event.Event
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
