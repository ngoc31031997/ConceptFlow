package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// CR-044 — the illustration library's tables.

const illustrationColumns = `id, name, title, folder_id, tags, description, usage, code, builtin, status, version,
	(preview_png IS NOT NULL AND preview_version = version), created_at, updated_at`

func scanIllustration(row pgx.Row) (domain.Illustration, error) {
	var i domain.Illustration
	var status string
	var created, updated time.Time
	if err := row.Scan(&i.ID, &i.Name, &i.Title, &i.FolderID, &i.Tags, &i.Description, &i.Usage, &i.Code,
		&i.Builtin, &status, &i.Version, &i.HasPreview, &created, &updated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Illustration{}, application.ErrIllustrationNotFound
		}
		return domain.Illustration{}, err
	}
	i.Status = domain.IllustrationStatus(status)
	if i.Tags == nil {
		i.Tags = []string{}
	}
	i.CreatedAt = created.Format(time.RFC3339)
	i.UpdatedAt = updated.Format(time.RFC3339)
	return i, nil
}

func mapIllustrationErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		case pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "illustrations_name"):
			return application.ErrIllustrationNameTaken
		case pgErr.Code == "23505":
			return application.ErrFolderTaken
		case pgErr.Code == "23503":
			return application.ErrFolderNotFound
		}
	}
	return err
}

// SeedIllustrations upserts the system folders and the built-in kit on every
// start, like SeedPrompts: a rebuilt image is the whole deploy. A Creator row
// that already took a built-in's name keeps it; the built-in is skipped.
func (r *PromptTemplateRepository) SeedIllustrations(ctx context.Context) error {
	for _, f := range domain.SystemIllustrationFolders() {
		if _, err := r.pool.Exec(ctx, `
			INSERT INTO illustration_folders (id, name, description, position, is_system)
			VALUES ($1, $2, $3, $4, true)
			ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description,
			    position = EXCLUDED.position, is_system = true
		`, f.ID, f.Name, f.Description, f.Position); err != nil {
			return err
		}
	}
	for _, b := range domain.BuiltinIllustrations() {
		if _, err := r.pool.Exec(ctx, `
			INSERT INTO illustrations (id, name, title, folder_id, tags, description, usage, builtin, status, version)
			VALUES ($1, $2, $3, $4, $5, $6, $7, true, 'approved', 1)
			ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title, folder_id = EXCLUDED.folder_id,
			    tags = EXCLUDED.tags, description = EXCLUDED.description, usage = EXCLUDED.usage,
			    updated_at = now()
		`, b.ID, b.Name, b.Title, b.FolderID, b.Tags, b.Description, b.Usage); err != nil {
			if errors.Is(mapIllustrationErr(err), application.ErrIllustrationNameTaken) {
				continue
			}
			return err
		}
	}
	return nil
}

func (r *PromptTemplateRepository) ListIllustrationFolders(ctx context.Context) ([]domain.IllustrationFolder, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, description, position, is_system FROM illustration_folders ORDER BY position, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.IllustrationFolder{}
	for rows.Next() {
		var f domain.IllustrationFolder
		if err := rows.Scan(&f.ID, &f.Name, &f.Description, &f.Position, &f.IsSystem); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *PromptTemplateRepository) CreateIllustrationFolder(ctx context.Context, f domain.IllustrationFolder) (domain.IllustrationFolder, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO illustration_folders (id, name, description, position, is_system)
		VALUES ($1, $2, $3, COALESCE((SELECT max(position) + 1 FROM illustration_folders), 1), false)
		RETURNING position
	`, f.ID, f.Name, f.Description).Scan(&f.Position)
	return f, mapIllustrationErr(err)
}

func (r *PromptTemplateRepository) DeleteIllustrationFolder(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM illustration_folders WHERE id = $1 AND NOT is_system`, id)
	return err
}

func (r *PromptTemplateRepository) ListIllustrations(ctx context.Context, f application.IllustrationFilter) ([]domain.Illustration, error) {
	q := "%" + strings.ToLower(strings.TrimSpace(f.Query)) + "%"
	rows, err := r.pool.Query(ctx, `
		SELECT `+illustrationColumns+` FROM illustrations
		WHERE ($1 = '' OR folder_id = $1)
		  AND ($2 = '' OR status = $2)
		  AND ($3 = '%%' OR lower(name) LIKE $3 OR lower(title) LIKE $3 OR lower(description) LIKE $3
		       OR lower(array_to_string(tags, ' ')) LIKE $3)
		ORDER BY builtin DESC, folder_id, title
	`, f.FolderID, string(f.Status), q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Illustration{}
	for rows.Next() {
		i, err := scanIllustration(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *PromptTemplateRepository) GetIllustration(ctx context.Context, id string) (domain.Illustration, error) {
	return scanIllustration(r.pool.QueryRow(ctx, `SELECT `+illustrationColumns+` FROM illustrations WHERE id = $1`, id))
}

func (r *PromptTemplateRepository) CreateIllustration(ctx context.Context, i domain.Illustration) (domain.Illustration, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO illustrations (name, title, folder_id, tags, description, usage, code, builtin, status, version)
		VALUES ($1, $2, $3, COALESCE($4::text[], '{}'), $5, $6, $7, false, $8, $9)
		RETURNING `+illustrationColumns, i.Name, i.Title, i.FolderID, i.Tags, i.Description, i.Usage, i.Code,
		string(i.Status), i.Version)
	out, err := scanIllustration(row)
	return out, mapIllustrationErr(err)
}

func (r *PromptTemplateRepository) UpdateIllustration(ctx context.Context, i domain.Illustration) (domain.Illustration, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE illustrations SET name = $2, title = $3, folder_id = $4, tags = COALESCE($5::text[], '{}'), description = $6, usage = $7,
		    code = $8, status = $9, version = $10, updated_at = now()
		WHERE id = $1 AND NOT builtin
		RETURNING `+illustrationColumns, i.ID, i.Name, i.Title, i.FolderID, i.Tags, i.Description, i.Usage, i.Code,
		string(i.Status), i.Version)
	out, err := scanIllustration(row)
	return out, mapIllustrationErr(err)
}

func (r *PromptTemplateRepository) DeleteIllustration(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM illustrations WHERE id = $1 AND NOT builtin`, id)
	return err
}

func (r *PromptTemplateRepository) SaveIllustrationPreview(ctx context.Context, id string, version int, png, gif []byte) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE illustrations SET preview_png = $3, preview_gif = $4, preview_version = $2 WHERE id = $1
	`, id, version, png, gif)
	return err
}

func (r *PromptTemplateRepository) GetIllustrationPreview(ctx context.Context, id string) ([]byte, []byte, error) {
	var png, gif []byte
	var current bool
	err := r.pool.QueryRow(ctx, `
		SELECT preview_png, preview_gif, preview_version = version FROM illustrations WHERE id = $1
	`, id).Scan(&png, &gif, &current)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, application.ErrIllustrationNotFound
	}
	if err != nil || !current {
		return nil, nil, err
	}
	return png, gif, nil
}
