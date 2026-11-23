package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"visual/internal/application"
)

// AIRunRepository implements application.AIRunStorePort over storyboard_ai_runs.
type AIRunRepository struct {
	pool *pgxpool.Pool
}

var _ application.AIRunStorePort = (*AIRunRepository)(nil)

// NewAIRunRepository returns the repository over pool.
func NewAIRunRepository(pool *pgxpool.Pool) *AIRunRepository {
	return &AIRunRepository{pool: pool}
}

const aiRunColumns = `id, project_id, task, target, note, channel, status, prompt, reply, preview, error, created_at, updated_at`

func scanAIRun(row pgx.Row) (application.AIRun, error) {
	var r application.AIRun
	var preview []byte
	err := row.Scan(&r.ID, &r.ProjectID, &r.Task, &r.Target, &r.Note, &r.Channel, &r.Status, &r.Prompt, &r.Reply,
		&preview, &r.Error, &r.CreatedAt, &r.UpdatedAt)
	if len(preview) > 0 {
		r.Preview = preview
	}
	return r, err
}

// nullJSON is the preview to write: NULL when there is none.
func nullJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}

// CreateAIRun stores a new run and returns it with its id and times.
func (r *AIRunRepository) CreateAIRun(ctx context.Context, run application.AIRun) (application.AIRun, error) {
	got, err := scanAIRun(r.pool.QueryRow(ctx, `
		INSERT INTO storyboard_ai_runs (project_id, task, target, note, channel, status, prompt, reply, preview, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10)
		RETURNING `+aiRunColumns,
		run.ProjectID, run.Task, run.Target, run.Note, run.Channel, run.Status, run.Prompt, run.Reply, nullJSON(run.Preview), run.Error))
	if err != nil {
		return application.AIRun{}, fmt.Errorf("create AI run: %w", err)
	}
	return got, nil
}

// AIRun returns one run of the video, or application.ErrUnknownAIRun.
func (r *AIRunRepository) AIRun(ctx context.Context, projectID, id string) (application.AIRun, error) {
	got, err := scanAIRun(r.pool.QueryRow(ctx,
		`SELECT `+aiRunColumns+` FROM storyboard_ai_runs WHERE project_id = $1 AND id = $2`, projectID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return application.AIRun{}, application.ErrUnknownAIRun
	}
	if err != nil {
		return application.AIRun{}, fmt.Errorf("read AI run: %w", err)
	}
	return got, nil
}

// SaveAIRun writes the run's status, reply, preview and error.
func (r *AIRunRepository) SaveAIRun(ctx context.Context, run application.AIRun) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE storyboard_ai_runs SET status = $3, reply = $4, preview = $5::jsonb, error = $6, updated_at = now()
		WHERE project_id = $1 AND id = $2`,
		run.ProjectID, run.ID, run.Status, run.Reply, nullJSON(run.Preview), run.Error)
	if err != nil {
		return fmt.Errorf("save AI run: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return application.ErrUnknownAIRun
	}
	return nil
}

// LatestAIRuns returns the newest run of every task and target of the video.
func (r *AIRunRepository) LatestAIRuns(ctx context.Context, projectID string) ([]application.AIRun, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ON (task, target) `+aiRunColumns+`
		FROM storyboard_ai_runs WHERE project_id = $1
		ORDER BY task, target, created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list AI runs: %w", err)
	}
	defer rows.Close()
	out := []application.AIRun{}
	for rows.Next() {
		run, err := scanAIRun(rows)
		if err != nil {
			return nil, fmt.Errorf("read AI run: %w", err)
		}
		out = append(out, run)
	}
	return out, rows.Err()
}

// FailRunningAIRuns marks every run still asking the model as failed with
// reason; it runs at start-up, when no model call of an earlier process can
// still finish.
func (r *AIRunRepository) FailRunningAIRuns(ctx context.Context, reason string) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE storyboard_ai_runs SET status = $1, error = $2, updated_at = now() WHERE status = $3`,
		application.AIRunFailed, reason, application.AIRunRunning)
	if err != nil {
		return 0, fmt.Errorf("fail running AI runs: %w", err)
	}
	return tag.RowsAffected(), nil
}
