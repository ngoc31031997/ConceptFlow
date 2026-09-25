package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"orchestrator/internal/application"
)

// maxProjectErrors bounds the column: a project stuck in a retry loop must not
// grow one row without limit. The newest entries are the useful ones.
const maxProjectErrors = 100

// AppendProjectError adds one entry to projects.project_errors, keeping the
// newest maxProjectErrors. A project that does not exist is a no-op (0 rows).
func (r *ProjectRepository) AppendProjectError(ctx context.Context, projectID string, e application.ProjectError) error {
	entry, err := json.Marshal([]application.ProjectError{e})
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE projects SET project_errors = COALESCE((
			SELECT jsonb_agg(x.e ORDER BY x.i)
			FROM (
				SELECT e, i FROM jsonb_array_elements(project_errors || $2::jsonb) WITH ORDINALITY AS t(e, i)
				ORDER BY i DESC LIMIT $3
			) x
		), '[]'::jsonb)
		WHERE project_id = $1`, projectID, string(entry), maxProjectErrors)
	return err
}

// ListProjectErrors returns the project's log, oldest first. Empty (never
// nil) for a project with no errors or that does not exist.
func (r *ProjectRepository) ListProjectErrors(ctx context.Context, projectID string) ([]application.ProjectError, error) {
	var raw []byte
	err := r.pool.QueryRow(ctx, `SELECT project_errors FROM projects WHERE project_id = $1`, projectID).Scan(&raw)
	out := []application.ProjectError{}
	if err != nil {
		// No such project — same answer as "no errors".
		if errors.Is(err, pgx.ErrNoRows) {
			return out, nil
		}
		return nil, err
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, err
		}
	}
	return out, nil
}
