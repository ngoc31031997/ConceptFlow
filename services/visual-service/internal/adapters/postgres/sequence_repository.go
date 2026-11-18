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

// SequenceRepository implements application.SequenceStorePort. Its review units
// are the shared review_units table and it reads the locked concept and plan
// through the plan repository.
type SequenceRepository struct {
	*PlanRepository
}

// NewSequenceRepository returns the repository over pool.
func NewSequenceRepository(pool *pgxpool.Pool) *SequenceRepository {
	return &SequenceRepository{PlanRepository: NewPlanRepository(pool)}
}

// LatestPlanVersion returns the newest locked visual plan with its document.
func (r *SequenceRepository) LatestPlanVersion(ctx context.Context, projectID string) (application.PlanVersionRecord, error) {
	v := application.PlanVersionRecord{ProjectID: projectID}
	err := r.pool.QueryRow(ctx, `
		SELECT version, script_version, concept_version, document, note, locked_at FROM visual_plan_versions
		WHERE project_id = $1 ORDER BY version DESC LIMIT 1`, projectID).
		Scan(&v.Version, &v.ScriptVersion, &v.ConceptVersion, &v.Document, &v.Note, &v.LockedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.PlanVersionRecord{}, domain.ErrNoVersion
	}
	if err != nil {
		return application.PlanVersionRecord{}, fmt.Errorf("read latest plan version: %w", err)
	}
	return v, nil
}

// SequenceVersions lists the locked sequence versions without documents, oldest first.
func (r *SequenceRepository) SequenceVersions(ctx context.Context, projectID string) ([]application.SequenceVersionRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT version, plan_version, concept_version, script_version, note, locked_at FROM visual_sequence_versions
		WHERE project_id = $1 ORDER BY version`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list sequence versions: %w", err)
	}
	defer rows.Close()
	var out []application.SequenceVersionRecord
	for rows.Next() {
		v := application.SequenceVersionRecord{ProjectID: projectID}
		if err := rows.Scan(&v.Version, &v.PlanVersion, &v.ConceptVersion, &v.ScriptVersion, &v.Note, &v.LockedAt); err != nil {
			return nil, fmt.Errorf("scan sequence version: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// SequenceVersion returns one locked sequence version with its document.
func (r *SequenceRepository) SequenceVersion(ctx context.Context, projectID string, version int) (application.SequenceVersionRecord, error) {
	v := application.SequenceVersionRecord{ProjectID: projectID, Version: version}
	err := r.pool.QueryRow(ctx, `
		SELECT plan_version, concept_version, script_version, document, note, locked_at FROM visual_sequence_versions
		WHERE project_id = $1 AND version = $2`, projectID, version).
		Scan(&v.PlanVersion, &v.ConceptVersion, &v.ScriptVersion, &v.Document, &v.Note, &v.LockedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.SequenceVersionRecord{}, domain.ErrNoVersion
	}
	if err != nil {
		return application.SequenceVersionRecord{}, fmt.Errorf("read sequence version: %w", err)
	}
	return v, nil
}

// InsertSequenceVersion adds a locked sequence version; versions are never changed.
func (r *SequenceRepository) InsertSequenceVersion(ctx context.Context, v application.SequenceVersionRecord) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO visual_sequence_versions (project_id, version, plan_version, concept_version, script_version, document, note, locked_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		v.ProjectID, v.Version, v.PlanVersion, v.ConceptVersion, v.ScriptVersion, []byte(v.Document), v.Note, v.LockedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrVersionExists
	}
	if err != nil {
		return fmt.Errorf("insert sequence version: %w", err)
	}
	return nil
}
