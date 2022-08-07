package rabbitmqtest

import (
	"context"

	"github.com/connorkuehl/popple/internal/event"
)

type EventRecorder struct {
	Events []*event.Event
}

func NewEventRecorder() *EventRecorder {
	return new(EventRecorder)
}

func (r *EventRecorder) PublishRequest(ctx context.Context, e *event.Event) error {
	r.Events = append(r.Events, e)
	return nil
}
