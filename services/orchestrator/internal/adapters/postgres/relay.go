package postgres

import (
	"context"
	"log/slog"
	"time"

	"orchestrator/internal/domain"
)

// relayBatchSize bounds how many outbox rows are fetched per poll.
const relayBatchSize = 50

// OutboxRelay is the background goroutine that turns queued commands into
// real AMQP publishes: poll outbox_events for published_at IS NULL rows,
// publish each via the injected domain.CommandPublisherPort (backed by
// amqp.Publisher — the one implementation that performs real network I/O),
// then mark it published (messaging-design.md "Relay Mechanism").
type OutboxRelay struct {
	outbox   *OutboxRepository
	sender   domain.CommandPublisherPort
	interval time.Duration
	logger   *slog.Logger
}

// NewOutboxRelay constructs the relay. interval is OUTBOX_POLL_INTERVAL_MS
// from config (default 500ms).
func NewOutboxRelay(outbox *OutboxRepository, sender domain.CommandPublisherPort, interval time.Duration, logger *slog.Logger) *OutboxRelay {
	if logger == nil {
		logger = slog.Default()
	}
	return &OutboxRelay{outbox: outbox, sender: sender, interval: interval, logger: logger}
}

// Run polls until ctx is cancelled. Intended to be started as
// `go relay.Run(ctx)` from main.go.
func (r *OutboxRelay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.pollOnce(ctx)
		}
	}
}

func (r *OutboxRelay) pollOnce(ctx context.Context) {
	pending, err := r.outbox.FetchUnpublished(ctx, relayBatchSize)
	if err != nil {
		r.logger.ErrorContext(ctx, "outbox relay: failed to fetch unpublished rows", "error", err)
		return
	}
	for _, row := range pending {
		if err := r.sender.PublishCommand(ctx, row.RoutingKey, row.Envelope); err != nil {
			r.logger.ErrorContext(ctx, "outbox relay: failed to publish command, will retry next poll", "error", err, "id", row.ID)
			continue
		}
		if err := r.outbox.MarkPublished(ctx, row.ID); err != nil {
			r.logger.ErrorContext(ctx, "outbox relay: failed to mark row published", "error", err, "id", row.ID)
		}
	}
}
