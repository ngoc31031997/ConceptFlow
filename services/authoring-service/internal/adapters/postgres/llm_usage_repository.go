package postgres

import (
	"context"
	"time"

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

// ModelUsageStats aggregates the calls of one step phase since a time, per
// model (CR-050 FR-19). Averages cover successful calls only: a failed call's
// duration says how long it took to fail, not what a chunk costs.
func (r *LLMUsageRepository) ModelUsageStats(
	ctx context.Context, step, phase string, since time.Time,
) ([]application.ModelUsageStats, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT model, ok, error_kind, count(*),
		       COALESCE(avg(completion_tokens), 0)::int, COALESCE(avg(duration_ms), 0)::int
		FROM llm_usage
		WHERE step = $1 AND phase = $2 AND created_at >= $3
		GROUP BY model, ok, error_kind
		ORDER BY model
	`, step, phase, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byModel := map[string]*application.ModelUsageStats{}
	var order []string
	for rows.Next() {
		var (
			model, kind          string
			ok                   bool
			n, avgOut, avgMillis int
		)
		if err := rows.Scan(&model, &ok, &kind, &n, &avgOut, &avgMillis); err != nil {
			return nil, err
		}
		st := byModel[model]
		if st == nil {
			st = &application.ModelUsageStats{Model: model, Failures: map[string]int{}}
			byModel[model] = st
			order = append(order, model)
		}
		st.Calls += n
		if ok {
			st.OK += n
			st.AvgCompletionTokens, st.AvgDurationMs = avgOut, avgMillis
		} else {
			if kind == "" {
				kind = "unknown"
			}
			st.Failures[kind] += n
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]application.ModelUsageStats, 0, len(order))
	for _, m := range order {
		out = append(out, *byModel[m])
	}
	return out, nil
}
