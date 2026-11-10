package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// projectTables are the tables that hold one video's data. reference_styles is
// shared by every video and is not one of them.
var projectTables = []string{"review_units", "visual_concept_versions", "project_reference_picks", "project_visual"}

// ProjectDataRepository implements application.ProjectDataPort.
type ProjectDataRepository struct {
	pool *pgxpool.Pool
}

// NewProjectDataRepository returns the repository over pool.
func NewProjectDataRepository(pool *pgxpool.Pool) *ProjectDataRepository {
	return &ProjectDataRepository{pool: pool}
}

// DeleteProjectData removes the project's rows from every per-video table in
// one transaction, so a failure leaves nothing half deleted.
func (r *ProjectDataRepository) DeleteProjectData(ctx context.Context, projectID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, table := range projectTables {
		if _, err := tx.Exec(ctx, "DELETE FROM "+table+" WHERE project_id = $1", projectID); err != nil {
			return fmt.Errorf("delete from %s: %w", table, err)
		}
	}
	return tx.Commit(ctx)
}
