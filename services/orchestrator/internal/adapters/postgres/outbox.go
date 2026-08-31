package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"orchestrator/internal/domain"
)

// OutboxRepository implements domain.CommandPublisherPort by ENQUEUING
// commands into outbox_events rather than publishing to AMQP directly — the
// actual send happens later, from OutboxRelay. This guarantees a command
// dispatch survives a crash between the Project/SagaStep state update and
// the network call to RabbitMQ (the crash-safety property this system's
// other units get from Outbox-for-events; ADR-0019 explains why Unit 8's
// Outbox instead holds commands, and why the table keeps the
// "outbox_events" name despite that semantic difference).
type OutboxRepository struct {
	pool *pgxpool.Pool
}

// NewOutboxRepository constructs the repository over an already-open pool.
func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

// PublishCommand enqueues envelope for later delivery by OutboxRelay,
// satisfying domain.CommandPublisherPort. Despite the name (required to
// match the interface), no network I/O happens here.
func (r *OutboxRepository) PublishCommand(ctx context.Context, routingKey string, envelope domain.Envelope) error {
	body, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO outbox_events (id, routing_key, payload) VALUES ($1, $2, $3)`,
		envelope.MessageID, routingKey, body)
	return err
}

// PendingRow is one unpublished outbox_events row, returned by
// FetchUnpublished for OutboxRelay to send.
type PendingRow struct {
	ID         string
	RoutingKey string
	Envelope   domain.Envelope
}

// FetchUnpublished returns up to limit rows with published_at IS NULL,
// oldest first — polled by OutboxRelay.
func (r *OutboxRepository) FetchUnpublished(ctx context.Context, limit int) ([]PendingRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, routing_key, payload FROM outbox_events
		WHERE published_at IS NULL ORDER BY created_at ASC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pending []PendingRow
	for rows.Next() {
		var id, routingKey string
		var payload []byte
		if err := rows.Scan(&id, &routingKey, &payload); err != nil {
			return nil, err
		}
		var envelope domain.Envelope
		if err := json.Unmarshal(payload, &envelope); err != nil {
			return nil, err
		}
		pending = append(pending, PendingRow{ID: id, RoutingKey: routingKey, Envelope: envelope})
	}
	return pending, rows.Err()
}

// MarkPublished sets published_at = now() for the given row id, after
// OutboxRelay has successfully sent it over AMQP.
func (r *OutboxRepository) MarkPublished(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE outbox_events SET published_at = now() WHERE id = $1`, id)
	return err
}
