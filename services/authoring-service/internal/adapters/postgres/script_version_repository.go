package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"authoring/internal/domain"
)

const scriptVersionColumns = `project_id, version, note, locked_at, parts`

func scanScriptVersion(row pgx.Row) (domain.ScriptVersion, error) {
	var v domain.ScriptVersion
	var raw []byte
	if err := row.Scan(&v.ProjectID, &v.Version, &v.Note, &v.LockedAt, &raw); err != nil {
		return domain.ScriptVersion{}, err
	}
	if err := json.Unmarshal(raw, &v.Parts); err != nil {
		return domain.ScriptVersion{}, fmt.Errorf("read script version parts: %w", err)
	}
	return v, nil
}

// SaveScriptVersion stores a new locked version. A version is never changed:
// saving one that already exists fails, so two locks at once cannot both win.
func (r *PromptTemplateRepository) SaveScriptVersion(ctx context.Context, v domain.ScriptVersion) error {
	raw, err := json.Marshal(v.Parts)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO script_versions (project_id, version, note, locked_at, parts) VALUES ($1, $2, $3, $4, $5)`,
		v.ProjectID, v.Version, v.Note, v.LockedAt, raw)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%w: phiên bản v%d đã được chốt ở nơi khác, tải lại màn", domain.ErrInvalidStatus, v.Version)
	}
	return err
}

// ListScriptVersions returns every locked version of a project, oldest first.
func (r *PromptTemplateRepository) ListScriptVersions(ctx context.Context, projectID string) ([]domain.ScriptVersion, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+scriptVersionColumns+` FROM script_versions WHERE project_id = $1 ORDER BY version`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ScriptVersion{}
	for rows.Next() {
		v, err := scanScriptVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// LatestScriptVersion returns the newest locked version, or domain.ErrNoLockedVersion.
func (r *PromptTemplateRepository) LatestScriptVersion(ctx context.Context, projectID string) (domain.ScriptVersion, error) {
	v, err := scanScriptVersion(r.pool.QueryRow(ctx,
		`SELECT `+scriptVersionColumns+` FROM script_versions WHERE project_id = $1 ORDER BY version DESC LIMIT 1`, projectID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ScriptVersion{}, domain.ErrNoLockedVersion
	}
	return v, err
}
