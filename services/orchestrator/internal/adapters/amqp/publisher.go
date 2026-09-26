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
	// controlExchange carries cancel requests to every worker. Declared here
	// (idempotent) as well as by the workers, so it exists whichever side
	// starts first and a running broker needs no definitions reload.
	controlExchange = "control.fanout"
)

// Publisher implements both domain.CommandPublisherPort and
// domain.ProgressPublisherPort (module-structure.md Interface Segregation —
// 2 separate interfaces, 1 implementing struct). It is used by
// postgres.OutboxRelay to perform the actual AMQP publish once a queued
// command has been picked up from outbox_events; use cases themselves talk
// to the Outbox-backed CommandPublisherPort implementation
// (postgres.OutboxRepository), not this struct directly, so a command
// dispatch survives a crash between the state update and the network call.
//
// It asks ConnectionManager for the live channel on every publish rather
// than caching one, so it keeps working across broker reconnects
// (ADR-0022).
type Publisher struct {
	chans *ConnectionManager
}

// NewPublisher wraps a ConnectionManager (constructed once in main.go —
// dependency-injection.md "Constructed directly").
func NewPublisher(chans *ConnectionManager) *Publisher {
	return &Publisher{chans: chans}
}

// PublishCommand publishes envelope to commands.direct with the given
// routing key (the target service's queue binding).
func (p *Publisher) PublishCommand(ctx context.Context, routingKey string, envelope domain.Envelope) error {
	body, err := EncodeEnvelope(envelope)
	if err != nil {
		return err
	}
	return p.chans.Channel().PublishWithContext(ctx, commandsExchange, routingKey, false, false, amqp.Publishing{
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
	return p.chans.Channel().PublishWithContext(ctx, progressExchange, "", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

// PublishCancel broadcasts a cancel request on control.fanout.
func (p *Publisher) PublishCancel(ctx context.Context, req domain.CancelRequest) error {
	body, err := json.Marshal(struct {
		Type string `json:"type"`
		domain.CancelRequest
	}{Type: "cancel", CancelRequest: req})
	if err != nil {
		return err
	}
	ch := p.chans.Channel()
	if err := ch.ExchangeDeclare(controlExchange, "fanout", true, false, false, false, nil); err != nil {
		return err
	}
	return ch.PublishWithContext(ctx, controlExchange, "", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}
