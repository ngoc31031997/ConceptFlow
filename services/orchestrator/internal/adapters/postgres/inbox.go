package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InboxRepository dedupes incoming event message_id values against
// processed_messages (Rule 9's "Inbox"), guarding against RabbitMQ
// redelivery causing HandleStepEventUseCase to run twice for the same
// event.
type InboxRepository struct {
	pool *pgxpool.Pool
}

// NewInboxRepository constructs the repository over an already-open pool.
func NewInboxRepository(pool *pgxpool.Pool) *InboxRepository {
	return &InboxRepository{pool: pool}
}

// HasProcessed reports whether messageID has already been recorded as
// processed.
func (r *InboxRepository) HasProcessed(ctx context.Context, messageID string) (bool, error) {
	var found string
	err := r.pool.QueryRow(ctx, `SELECT message_id FROM processed_messages WHERE message_id = $1`, messageID).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// MarkProcessed records messageID as processed.
func (r *InboxRepository) MarkProcessed(ctx context.Context, messageID string) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO processed_messages (message_id) VALUES ($1) ON CONFLICT (message_id) DO NOTHING`, messageID)
	return err
}
