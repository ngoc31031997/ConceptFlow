package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"authoring/internal/application"
)

// LLMUsageRepository implements application.LLMUsagePort against the
// llm_usage table (CR-027 D9).
type LLMUsageRepository struct {
	pool *pgxpool.Pool
}

// NewLLMUsageRepository constructs the repository over an already-open pool.
func NewLLMUsageRepository(pool *pgxpool.Pool) *LLMUsageRepository {
	return &LLMUsageRepository{pool: pool}
}

// RecordLLMUsage appends one call.
//
// Append-only, like QCReportRepository and for a related reason: this is a
// ledger. A second call for the same project/step is a second charge, not a
// correction of the first, and an upsert would quietly make spend look lower
// than the invoice.
//
// An empty ProjectID is written as SQL NULL rather than ”: the column means
// "which project this served", and a call that served none is genuinely
// unknown, not a project whose id happens to be blank.
func (r *LLMUsageRepository) RecordLLMUsage(ctx context.Context, rec application.LLMUsageRecord) error {
	var projectID any
	if rec.ProjectID != "" {
		projectID = rec.ProjectID
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO llm_usage (
			provider, model, role, step, project_id,
			prompt_tokens, completion_tokens, reasoning_tokens, cached_tokens,
			duration_ms, ok, error_kind, phase
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`,
		rec.Provider, rec.Model, rec.Role, rec.Step, projectID,
		rec.PromptTokens, rec.CompletionTokens, rec.ReasoningTokens, rec.CachedTokens,
		rec.Duration.Milliseconds(), rec.OK, string(rec.ErrorKind), rec.Phase,
	)
	return err
}
