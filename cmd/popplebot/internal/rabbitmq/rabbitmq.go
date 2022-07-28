package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/connorkuehl/popple/internal/event"
)

type RequestPublisher struct {
	ch   *amqp.Channel
	name string
}

func NewRequestPublisher(ch *amqp.Channel, name string) *RequestPublisher {
	return &RequestPublisher{
		ch:   ch,
		name: name,
	}
}

func (p *RequestPublisher) PublishRequest(ctx context.Context, req *event.Event) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to serialize request: %w", err)
	}

	return p.ch.PublishWithContext(
		ctx,
		"",
		p.name,
		false,
		false,
		amqp.Publishing{
			Body: payload,
		},
	)
}
