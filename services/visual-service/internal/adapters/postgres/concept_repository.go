package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"visual/internal/application"
	"visual/internal/domain"
)

// ConceptRepository implements application.ConceptStorePort and application.ReferenceReader.
type ConceptRepository struct {
	pool *pgxpool.Pool
}

// NewConceptRepository returns the repository over pool.
func NewConceptRepository(pool *pgxpool.Pool) *ConceptRepository {
	return &ConceptRepository{pool: pool}
}

// ListUnits returns the project's units of kind.
func (r *ConceptRepository) ListUnits(ctx context.Context, projectID, kind string) ([]domain.ReviewUnit, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT project_id, kind, unit_key, status, content, creator_fields, input_hashes, redo_count, created_at, updated_at
		FROM review_units WHERE project_id = $1 AND kind = $2`, projectID, kind)
	if err != nil {
		return nil, fmt.Errorf("list units: %w", err)
	}
	defer rows.Close()
	var out []domain.ReviewUnit
	for rows.Next() {
		var u domain.ReviewUnit
		var hashes []byte
		if err := rows.Scan(&u.ProjectID, &u.Kind, &u.Key, &u.Status, &u.Content, &u.CreatorFields, &hashes, &u.RedoCount, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan unit: %w", err)
		}
		if err := json.Unmarshal(hashes, &u.InputHashes); err != nil {
			return nil, fmt.Errorf("read input_hashes: %w", err)
		}
		if string(u.Content) == "{}" {
			u.Content = nil
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// SaveUnit inserts or replaces a unit.
func (r *ConceptRepository) SaveUnit(ctx context.Context, u domain.ReviewUnit) error {
	hashes, err := json.Marshal(u.InputHashes)
	if err != nil {
		return err
	}
	content := u.Content
	if len(content) == 0 {
		content = []byte("{}")
	}
	fields := u.CreatorFields
	if fields == nil {
		fields = []string{}
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO review_units (project_id, kind, unit_key, status, content, creator_fields, input_hashes, redo_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (project_id, kind, unit_key) DO UPDATE SET
		  status = EXCLUDED.status, content = EXCLUDED.content, creator_fields = EXCLUDED.creator_fields,
		  input_hashes = EXCLUDED.input_hashes, redo_count = EXCLUDED.redo_count, updated_at = now()`,
		u.ProjectID, u.Kind, u.Key, u.Status, content, fields, hashes, u.RedoCount)
	if err != nil {
		return fmt.Errorf("save unit: %w", err)
	}
	return nil
}

// ConceptModel returns the video's model of step 8; ” when none is saved.
func (r *ConceptRepository) ConceptModel(ctx context.Context, projectID string) (string, error) {
	var m string
	err := r.pool.QueryRow(ctx, `SELECT concept_model FROM project_visual WHERE project_id = $1`, projectID).Scan(&m)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read concept model: %w", err)
	}
	return m, nil
}

// SetConceptModel saves the video's model of step 8.
func (r *ConceptRepository) SetConceptModel(ctx context.Context, projectID, model string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO project_visual (project_id, concept_model) VALUES ($1, $2)
		ON CONFLICT (project_id) DO UPDATE SET concept_model = EXCLUDED.concept_model, updated_at = now()`, projectID, model)
	if err != nil {
		return fmt.Errorf("save concept model: %w", err)
	}
	return nil
}

// refColumns are the columns of a reference description; detail holds every
// field but the name, intent and tags.
const refColumns = `s.id, s.name, s.description, s.tags, s.detail`

func scanRefs(rows pgx.Rows) ([]domain.PickedReference, error) {
	defer rows.Close()
	out := []domain.PickedReference{}
	for rows.Next() {
		var s domain.PickedReference
		var detail []byte
		if err := rows.Scan(&s.ID, &s.StyleName, &s.Intent, &s.Tags, &detail); err != nil {
			return nil, fmt.Errorf("scan reference: %w", err)
		}
		if err := applyDetail(detail, &s.ReferenceStyleInput); err != nil {
			return nil, fmt.Errorf("read detail of reference %s: %w", s.ID, err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// PickedReferences returns the descriptions the video picked, in library order.
func (r *ConceptRepository) PickedReferences(ctx context.Context, projectID string) ([]domain.PickedReference, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+refColumns+` FROM project_reference_picks p
		JOIN reference_styles s ON s.id = p.ref_id WHERE p.project_id = $1 ORDER BY s.seq`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list picks: %w", err)
	}
	return scanRefs(rows)
}

// ReferenceLibrary returns every description of the shared library.
func (r *ConceptRepository) ReferenceLibrary(ctx context.Context) ([]domain.PickedReference, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+refColumns+` FROM reference_styles s ORDER BY s.seq`)
	if err != nil {
		return nil, fmt.Errorf("list references: %w", err)
	}
	return scanRefs(rows)
}

// SetPicks replaces the video's picks in one transaction.
func (r *ConceptRepository) SetPicks(ctx context.Context, projectID string, refIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if len(refIDs) > 0 {
		var n int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM reference_styles WHERE id = ANY($1)`, refIDs).Scan(&n); err != nil {
			return fmt.Errorf("check references: %w", err)
		}
		if n != len(refIDs) {
			return domain.ErrUnknownReference
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM project_reference_picks WHERE project_id = $1`, projectID); err != nil {
		return fmt.Errorf("clear picks: %w", err)
	}
	for _, id := range refIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO project_reference_picks (project_id, ref_id) VALUES ($1, $2)`, projectID, id); err != nil {
			return fmt.Errorf("pick %s: %w", id, err)
		}
	}
	return tx.Commit(ctx)
}

// Versions lists the locked versions without documents, oldest first.
func (r *ConceptRepository) Versions(ctx context.Context, projectID string) ([]application.VersionRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT version, script_version, note, locked_at FROM visual_concept_versions
		WHERE project_id = $1 ORDER BY version`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()
	var out []application.VersionRecord
	for rows.Next() {
		v := application.VersionRecord{ProjectID: projectID}
		if err := rows.Scan(&v.Version, &v.ScriptVersion, &v.Note, &v.LockedAt); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// Version returns one locked version with its document.
func (r *ConceptRepository) Version(ctx context.Context, projectID string, version int) (application.VersionRecord, error) {
	v := application.VersionRecord{ProjectID: projectID, Version: version}
	err := r.pool.QueryRow(ctx, `
		SELECT script_version, document, note, locked_at FROM visual_concept_versions
		WHERE project_id = $1 AND version = $2`, projectID, version).Scan(&v.ScriptVersion, &v.Document, &v.Note, &v.LockedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.VersionRecord{}, domain.ErrNoVersion
	}
	if err != nil {
		return application.VersionRecord{}, fmt.Errorf("read version: %w", err)
	}
	return v, nil
}

// InsertVersion adds a locked version; versions are never changed.
func (r *ConceptRepository) InsertVersion(ctx context.Context, v application.VersionRecord) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO visual_concept_versions (project_id, version, script_version, document, note, locked_at)
		VALUES ($1, $2, $3, $4, $5, $6)`, v.ProjectID, v.Version, v.ScriptVersion, []byte(v.Document), v.Note, v.LockedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrVersionExists
	}
	if err != nil {
		return fmt.Errorf("insert version: %w", err)
	}
	return nil
}
