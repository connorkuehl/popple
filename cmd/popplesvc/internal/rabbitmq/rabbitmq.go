package rabbitmq

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/connorkuehl/popple/event"
)

type EventBus struct {
	ch *amqp.Channel
}

func NewEventBus(ch *amqp.Channel) *EventBus {
	return &EventBus{
		ch: ch,
	}
}

func (e *EventBus) EmitEvent(ctx context.Context, key string, evt *event.Event) error {
	payload, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	// TODO: "popple_topic" should be a global and accessible by both bot and svc pkgs.
	return e.ch.PublishWithContext(
		ctx,
		"popple_topic", // exchange
		key,            //routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			Body: payload,
		},
	)
}
