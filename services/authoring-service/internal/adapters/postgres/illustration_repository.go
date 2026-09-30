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

// The illustration library's tables.

const illustrationColumns = `id, name, title, folder_id, tags, description, usage, code, builtin, exemplar,
	COALESCE(source_id, ''), COALESCE(home_folder_id, ''), warnings, status, version,
	(preview_png IS NOT NULL AND preview_version = version), created_at, updated_at`

func scanIllustration(row pgx.Row) (domain.Illustration, error) {
	var i domain.Illustration
	var status string
	var created, updated time.Time
	if err := row.Scan(&i.ID, &i.Name, &i.Title, &i.FolderID, &i.Tags, &i.Description, &i.Usage, &i.Code,
		&i.Builtin, &i.Exemplar, &i.SourceID, &i.HomeFolderID, &i.Warnings, &status, &i.Version, &i.HasPreview, &created, &updated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Illustration{}, application.ErrIllustrationNotFound
		}
		return domain.Illustration{}, err
	}
	i.Status = domain.IllustrationStatus(status)
	if i.Tags == nil {
		i.Tags = []string{}
	}
	if i.Warnings == nil {
		i.Warnings = []domain.CodeFinding{}
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
	// The three original exemplars become Hình mẫu
	// data, once. After that they are rows like any other: never seeded again,
	// so one the Creator took out of the Hình mẫu stays out.
	if _, err := r.pool.Exec(ctx, `
		UPDATE illustrations SET home_folder_id = folder_id, folder_id = $1, builtin = false, updated_at = now()
		WHERE exemplar AND builtin
	`, domain.ExemplarFolderID); err != nil {
		return err
	}
	// The kit.
	for _, b := range domain.BuiltinIllustrations() {
		if _, err := r.pool.Exec(ctx, `
			INSERT INTO illustrations (id, name, title, folder_id, tags, description, usage, code, builtin, exemplar, status, version)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, $9, 'approved', 1)
			ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title, folder_id = EXCLUDED.folder_id,
			    tags = EXCLUDED.tags, description = EXCLUDED.description, usage = EXCLUDED.usage,
			    exemplar = EXCLUDED.exemplar, code = EXCLUDED.code,
			    version = illustrations.version + (illustrations.code IS DISTINCT FROM EXCLUDED.code)::int,
			    updated_at = now()
		`, b.ID, b.Name, b.Title, b.FolderID, b.Tags, b.Description, b.Usage, b.Code, b.Exemplar); err != nil {
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
		INSERT INTO illustrations (name, title, folder_id, tags, description, usage, code, builtin, status, version, warnings)
		VALUES ($1, $2, $3, COALESCE($4::text[], '{}'), $5, $6, $7, false, $8, $9, $10)
		RETURNING `+illustrationColumns, i.Name, i.Title, i.FolderID, i.Tags, i.Description, i.Usage, i.Code,
		string(i.Status), i.Version, findings(i.Warnings))
	out, err := scanIllustration(row)
	return out, mapIllustrationErr(err)
}

func (r *PromptTemplateRepository) UpdateIllustration(ctx context.Context, i domain.Illustration) (domain.Illustration, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE illustrations SET name = $2, title = $3, folder_id = $4, tags = COALESCE($5::text[], '{}'), description = $6, usage = $7,
		    code = $8, status = $9, version = $10, warnings = $11, updated_at = now()
		WHERE id = $1 AND NOT builtin
		RETURNING `+illustrationColumns, i.ID, i.Name, i.Title, i.FolderID, i.Tags, i.Description, i.Usage, i.Code,
		string(i.Status), i.Version, findings(i.Warnings))
	out, err := scanIllustration(row)
	return out, mapIllustrationErr(err)
}

// DeleteIllustration refuses (ErrIllustrationLinked) when a project outside
// checked links the drawing: one that planned it after the caller looked.
func (r *PromptTemplateRepository) DeleteIllustration(ctx context.Context, id string, checked []string) error {
	if checked == nil {
		checked = []string{}
	}
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM illustrations WHERE id = $1 AND NOT builtin
		  AND NOT EXISTS (SELECT 1 FROM project_illustrations WHERE illustration_id = $1 AND project_id <> ALL($2::text[]))
	`, id, checked)
	if err != nil || tag.RowsAffected() > 0 {
		return err
	}
	var linked bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM project_illustrations WHERE illustration_id = $1 AND project_id <> ALL($2::text[]))
	`, id, checked).Scan(&linked); err != nil {
		return err
	}
	if linked {
		return application.ErrIllustrationLinked
	}
	return nil
}

// FindIllustrationUsers lists the projects whose drawing list links the
// drawing or whose saved code names its component (a whole word: Bus is not
// BusStop). Names are [A-Za-z0-9] only, so they are safe inside the regex.
func (r *PromptTemplateRepository) FindIllustrationUsers(ctx context.Context, id, name string) ([]application.IllustrationUser, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.project_id, COALESCE(pa.topic, '') FROM (
		    SELECT project_id FROM project_illustrations WHERE illustration_id = $1
		    UNION
		    SELECT project_id FROM project_authoring WHERE code_content ~ ('\m' || $2 || '\M')
		) u LEFT JOIN project_authoring pa ON pa.project_id = u.project_id
		ORDER BY u.project_id
	`, id, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []application.IllustrationUser{}
	for rows.Next() {
		var u application.IllustrationUser
		if err := rows.Scan(&u.ProjectID, &u.Topic); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// CreateExemplar inserts a Hình mẫu. The Hình mẫu folder row is locked for
// the count, so two "Đặt làm mẫu" at once cannot pass the limit together.
func (r *PromptTemplateRepository) CreateExemplar(ctx context.Context, i domain.Illustration, limit int) (domain.Illustration, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return i, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT 1 FROM illustration_folders WHERE id = $1 FOR UPDATE`, domain.ExemplarFolderID); err != nil {
		return i, err
	}
	var count int
	var copied bool
	if err := tx.QueryRow(ctx, `
		SELECT count(*), COALESCE(bool_or(source_id = NULLIF($1, '')), false) FROM illustrations WHERE exemplar
	`, i.SourceID).Scan(&count, &copied); err != nil {
		return i, err
	}
	if copied {
		return i, application.ErrAlreadyExemplar
	}
	if count >= limit {
		return i, application.ErrExemplarLimit
	}
	out, err := scanIllustration(tx.QueryRow(ctx, `
		INSERT INTO illustrations (name, title, folder_id, tags, description, usage, code, builtin, exemplar,
		    source_id, home_folder_id, status, version, warnings)
		VALUES ($1, $2, $3, COALESCE($4::text[], '{}'), $5, $6, $7, false, true, NULLIF($8, ''), NULLIF($9, ''), $10, $11, $12)
		RETURNING `+illustrationColumns, i.Name, i.Title, domain.ExemplarFolderID, i.Tags, i.Description, i.Usage, i.Code,
		i.SourceID, i.HomeFolderID, string(i.Status), i.Version, findings(i.Warnings)))
	if err != nil {
		return out, mapIllustrationErr(err)
	}
	return out, tx.Commit(ctx)
}

// ReleaseExemplar files an original exemplar back into folderID as an
// ordinary drawing, keeping its id, name and approval.
func (r *PromptTemplateRepository) ReleaseExemplar(ctx context.Context, id, folderID string) (domain.Illustration, error) {
	out, err := scanIllustration(r.pool.QueryRow(ctx, `
		UPDATE illustrations SET exemplar = false, home_folder_id = NULL, folder_id = $2, updated_at = now()
		WHERE id = $1 AND exemplar
		RETURNING `+illustrationColumns, id, folderID))
	return out, mapIllustrationErr(err)
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

// findings stores a nil slice as [] so the column's NOT NULL holds.
func findings(f []domain.CodeFinding) []domain.CodeFinding {
	if f == nil {
		return []domain.CodeFinding{}
	}
	return f
}
