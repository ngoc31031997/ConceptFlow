package amqp

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"

	"orchestrator/internal/domain"
)

const (
	commandsExchange = "commands.direct"
	progressExchange = "progress.fanout"
)

// Publisher implements both domain.CommandPublisherPort and
// domain.ProgressPublisherPort (module-structure.md Interface Segregation —
// 2 separate interfaces, 1 implementing struct). It is used by
// postgres.OutboxRelay to perform the actual AMQP publish once a queued
// command has been picked up from outbox_events; use cases themselves talk
// to the Outbox-backed CommandPublisherPort implementation
// (postgres.OutboxRepository), not this struct directly, so a command
// dispatch survives a crash between the state update and the network call.
type Publisher struct {
	channel *amqp.Channel
}

// NewPublisher wraps an already-open amqp091-go channel (constructed once in
// main.go — dependency-injection.md "Constructed directly").
func NewPublisher(channel *amqp.Channel) *Publisher {
	return &Publisher{channel: channel}
}

// PublishCommand publishes envelope to commands.direct with the given
// routing key (the target service's queue binding).
func (p *Publisher) PublishCommand(ctx context.Context, routingKey string, envelope domain.Envelope) error {
	body, err := EncodeEnvelope(envelope)
	if err != nil {
		return err
	}
	return p.channel.PublishWithContext(ctx, commandsExchange, routingKey, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

// PublishProgress publishes msg to progress.fanout (ADR-0017). Fire-and-
// forget from the caller's perspective — dependency-injection.md's
// Concurrency Model notes this must not block the event-processing
// goroutine.
func (p *Publisher) PublishProgress(ctx context.Context, msg domain.ProgressMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.channel.PublishWithContext(ctx, progressExchange, "", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}
