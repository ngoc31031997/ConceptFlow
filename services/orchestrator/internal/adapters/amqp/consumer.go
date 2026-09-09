package amqp

import (
	"context"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"

	"orchestrator/internal/application"
	"orchestrator/internal/domain"
)

// eventsQueue is the single queue Orchestrator consumes all 12 Saga event
// types from (interface-contracts.md).
const eventsQueue = "orchestrator.events"

// dlqQueues are the 6 dead-letter queues (one per command routing key,
// Unit 1's "*.commands.dlq" pattern) Orchestrator also consumes — a
// dead-lettered command means a downstream service never picked it up.
var dlqQueues = []string{
	"script_processing.commands.dlq",
	"tts.commands.dlq",
	"rendering.commands.dlq",
	"video_assembly.commands.dlq",
	"publisher.commands.dlq",
}

// eventHandler is the minimal capability consumer.go needs from
// application.HandleStepEventUseCase.
type eventHandler interface {
	Execute(ctx context.Context, event application.StepEvent) error
}

// inboxPort is the minimal capability consumer.go needs from
// postgres.InboxRepository — dedupe of incoming message_id (Rule 9).
type inboxPort interface {
	HasProcessed(ctx context.Context, messageID string) (bool, error)
	MarkProcessed(ctx context.Context, messageID string) error
}

// Consumer wires RabbitMQ deliveries to HandleStepEventUseCase, deduping via
// the Inbox and spawning one goroutine per message
// (dependency-injection.md "Concurrency Model").
//
// It re-issues its Consume registrations against ConnectionManager after
// every reconnect (see Start and main.go's OnReconnect wiring) — a broker
// reconnect implicitly drops all consumers, and the old delivery loops
// below already exit on their own when their Go channel closes with the
// dead AMQP channel, so Start is safe to call again with no explicit
// teardown (ADR-0022).
type Consumer struct {
	chans   *ConnectionManager
	inbox   inboxPort
	handler eventHandler
	repo    domain.ProjectRepositoryPort
	logger  *slog.Logger
}

// NewConsumer constructs the Consumer.
func NewConsumer(chans *ConnectionManager, inbox inboxPort, handler eventHandler, repo domain.ProjectRepositoryPort, logger *slog.Logger) *Consumer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Consumer{chans: chans, inbox: inbox, handler: handler, repo: repo, logger: logger}
}

// Start begins consuming orchestrator.events and all 6 DLQ queues. It
// returns once all consumers are registered; delivery handling happens in
// background goroutines for the lifetime of the channel. Safe to call again
// after a reconnect (see type doc).
func (c *Consumer) Start(ctx context.Context) error {
	if err := c.consumeEvents(ctx, eventsQueue); err != nil {
		return err
	}
	for _, q := range dlqQueues {
		if err := c.consumeDLQ(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

func (c *Consumer) consumeEvents(ctx context.Context, queue string) error {
	deliveries, err := c.chans.Channel().Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for d := range deliveries {
			delivery := d
			go c.handleEventDelivery(ctx, delivery)
		}
	}()
	return nil
}

func (c *Consumer) consumeDLQ(ctx context.Context, queue string) error {
	deliveries, err := c.chans.Channel().Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for d := range deliveries {
			delivery := d
			go c.handleDLQDelivery(ctx, delivery)
		}
	}()
	return nil
}

// resolveEventType extracts the event type from an inbound event envelope.
// The 6 upstream Python services publish events per Unit 1's approved
// envelope standard (messaging-design.md), which carries "event_type"
// INSIDE payload, not as a top-level envelope field — unlike the commands
// Orchestrator itself publishes (application/dispatch), which use a
// top-level EventType (a Go-only convention no other service reads, since
// each command queue is already dedicated to one command type). Found via
// live E2E testing: reading envelope.EventType for an inbound event always
// came back empty, so every single event was silently dropped as "unknown
// event_type" and every Saga hung forever.
func resolveEventType(envelope domain.Envelope) string {
	if et, ok := envelope.Payload["event_type"].(string); ok && et != "" {
		return et
	}
	return envelope.EventType
}

func (c *Consumer) handleEventDelivery(ctx context.Context, d amqp.Delivery) {
	envelope, err := DecodeEnvelope(d.Body)
	if err != nil {
		c.logger.ErrorContext(ctx, "failed to decode event envelope, dropping", "error", err)
		_ = d.Ack(false)
		return
	}

	processed, err := c.inbox.HasProcessed(ctx, envelope.MessageID)
	if err != nil {
		c.logger.ErrorContext(ctx, "inbox check failed, nacking for redelivery", "error", err, "message_id", envelope.MessageID)
		_ = d.Nack(false, true)
		return
	}
	if processed {
		// Message-level idempotency (Rule 9) — already handled, ack and drop.
		_ = d.Ack(false)
		return
	}

	eventType := resolveEventType(envelope)

	err = c.handler.Execute(ctx, application.StepEvent{
		MessageID: envelope.MessageID,
		SagaID:    envelope.SagaID,
		ProjectID: envelope.ProjectID,
		EventType: eventType,
		Payload:   envelope.Payload,
	})
	if err != nil {
		c.logger.ErrorContext(ctx, "event processing failed, nacking for redelivery", "error", err, "event_type", eventType)
		_ = d.Nack(false, true)
		return
	}

	if err := c.inbox.MarkProcessed(ctx, envelope.MessageID); err != nil {
		c.logger.ErrorContext(ctx, "failed to mark message processed", "error", err, "message_id", envelope.MessageID)
	}
	_ = d.Ack(false)
}

// handleDLQDelivery handles a dead-lettered command: the envelope's
// EventType is the step name it was originally dispatched for (commands are
// published with EventType = step name, see application's dispatch
// helpers), so no queue-name-to-step lookup table is needed.
func (c *Consumer) handleDLQDelivery(ctx context.Context, d amqp.Delivery) {
	envelope, err := DecodeEnvelope(d.Body)
	if err != nil {
		c.logger.ErrorContext(ctx, "failed to decode DLQ envelope, dropping", "error", err)
		_ = d.Ack(false)
		return
	}

	stepName := domain.StepName(envelope.EventType)
	errMsg := "message dead-lettered: exceeded delivery limit"
	if err := c.repo.UpdateStep(ctx, &domain.SagaStep{
		SagaID: envelope.SagaID, StepName: stepName, Status: domain.SagaStepFailed, ErrorMessage: &errMsg,
	}); err != nil {
		c.logger.ErrorContext(ctx, "failed to update saga step for DLQ delivery", "error", err)
		_ = d.Nack(false, true)
		return
	}
	if err := c.repo.UpdateStatus(ctx, envelope.ProjectID, domain.FailedStatusForStep(stepName)); err != nil {
		c.logger.ErrorContext(ctx, "failed to update project status for DLQ delivery", "error", err)
		_ = d.Nack(false, true)
		return
	}
	_ = d.Ack(false)
}
