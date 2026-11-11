package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"visual/internal/application"
	"visual/internal/domain"
)

// PlanRepository implements application.PlanStorePort. Its review units are
// the shared review_units table, read through the concept repository.
type PlanRepository struct {
	*ConceptRepository
}

// NewPlanRepository returns the repository over pool.
func NewPlanRepository(pool *pgxpool.Pool) *PlanRepository {
	return &PlanRepository{ConceptRepository: NewConceptRepository(pool)}
}

// ReserveUnitNumbers hands out n consecutive numbers in one statement, so two
// requests never get the same one.
func (r *PlanRepository) ReserveUnitNumbers(ctx context.Context, projectID string, n int) (int, error) {
	if n < 1 {
		return 0, fmt.Errorf("reserve %d unit numbers: need at least one", n)
	}
	var first int
	err := r.pool.QueryRow(ctx, `
		INSERT INTO visual_plan_seq (project_id, next) VALUES ($1, $2::int + 1)
		ON CONFLICT (project_id) DO UPDATE SET next = visual_plan_seq.next + $2::int
		RETURNING next - $2::int`, projectID, n).Scan(&first)
	if err != nil {
		return 0, fmt.Errorf("reserve unit numbers: %w", err)
	}
	return first, nil
}

// LatestConceptVersion returns the newest locked visual concept with its document.
func (r *PlanRepository) LatestConceptVersion(ctx context.Context, projectID string) (application.VersionRecord, error) {
	v := application.VersionRecord{ProjectID: projectID}
	err := r.pool.QueryRow(ctx, `
		SELECT version, script_version, document, note, locked_at FROM visual_concept_versions
		WHERE project_id = $1 ORDER BY version DESC LIMIT 1`, projectID).Scan(&v.Version, &v.ScriptVersion, &v.Document, &v.Note, &v.LockedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.VersionRecord{}, domain.ErrNoVersion
	}
	if err != nil {
		return application.VersionRecord{}, fmt.Errorf("read latest concept version: %w", err)
	}
	return v, nil
}

// PlanVersions lists the locked plan versions without documents, oldest first.
func (r *PlanRepository) PlanVersions(ctx context.Context, projectID string) ([]application.PlanVersionRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT version, script_version, concept_version, note, locked_at FROM visual_plan_versions
		WHERE project_id = $1 ORDER BY version`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list plan versions: %w", err)
	}
	defer rows.Close()
	var out []application.PlanVersionRecord
	for rows.Next() {
		v := application.PlanVersionRecord{ProjectID: projectID}
		if err := rows.Scan(&v.Version, &v.ScriptVersion, &v.ConceptVersion, &v.Note, &v.LockedAt); err != nil {
			return nil, fmt.Errorf("scan plan version: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// PlanVersion returns one locked plan version with its document.
func (r *PlanRepository) PlanVersion(ctx context.Context, projectID string, version int) (application.PlanVersionRecord, error) {
	v := application.PlanVersionRecord{ProjectID: projectID, Version: version}
	err := r.pool.QueryRow(ctx, `
		SELECT script_version, concept_version, document, note, locked_at FROM visual_plan_versions
		WHERE project_id = $1 AND version = $2`, projectID, version).Scan(&v.ScriptVersion, &v.ConceptVersion, &v.Document, &v.Note, &v.LockedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.PlanVersionRecord{}, domain.ErrNoVersion
	}
	if err != nil {
		return application.PlanVersionRecord{}, fmt.Errorf("read plan version: %w", err)
	}
	return v, nil
}

// InsertPlanVersion adds a locked plan version; versions are never changed.
func (r *PlanRepository) InsertPlanVersion(ctx context.Context, v application.PlanVersionRecord) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO visual_plan_versions (project_id, version, script_version, concept_version, document, note, locked_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, v.ProjectID, v.Version, v.ScriptVersion, v.ConceptVersion, []byte(v.Document), v.Note, v.LockedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrVersionExists
	}
	if err != nil {
		return fmt.Errorf("insert plan version: %w", err)
	}
	return nil
}
