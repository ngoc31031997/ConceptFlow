package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"authoring/internal/domain"
)

// visualLanguageSelect reads a language joined with its newest version.
const visualLanguageSelect = `
	SELECT l.id, l.name, l.note, l.description, l.kit_key, l.is_system, l.is_active,
	       v.version, v.illustration_rules, v.backdrop_rules, v.scene_rules, l.created_at
	FROM visual_languages l
	JOIN visual_language_versions v
	  ON v.language_id = l.id
	 AND v.version = (SELECT max(version) FROM visual_language_versions WHERE language_id = l.id)`

func scanVisualLanguage(row pgx.Row) (domain.VisualLanguage, error) {
	var l domain.VisualLanguage
	var created time.Time
	err := row.Scan(&l.ID, &l.Name, &l.Note, &l.Description, &l.KitKey, &l.IsSystem, &l.IsActive,
		&l.Version, &l.Rules.Illustration, &l.Rules.Backdrop, &l.Rules.Scene, &created)
	l.CreatedAt = created.UTC().Format(time.RFC3339)
	return l, err
}

// ListVisualLanguages lists every language at its newest version, the active
// one first, then the shipped ones, then the Creator's by name.
func (r *PromptTemplateRepository) ListVisualLanguages(ctx context.Context) ([]domain.VisualLanguage, error) {
	rows, err := r.pool.Query(ctx, visualLanguageSelect+` ORDER BY l.is_active DESC, l.is_system DESC, l.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.VisualLanguage
	for rows.Next() {
		l, err := scanVisualLanguage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// GetVisualLanguage returns one language at its newest version.
func (r *PromptTemplateRepository) GetVisualLanguage(ctx context.Context, id string) (domain.VisualLanguage, error) {
	l, err := scanVisualLanguage(r.pool.QueryRow(ctx, visualLanguageSelect+` WHERE l.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return l, domain.ErrVisualLanguageNotFound
	}
	return l, err
}

// ActiveVisualLanguage returns the active language.
func (r *PromptTemplateRepository) ActiveVisualLanguage(ctx context.Context) (domain.VisualLanguage, error) {
	l, err := scanVisualLanguage(r.pool.QueryRow(ctx, visualLanguageSelect+` WHERE l.is_active`))
	if errors.Is(err, pgx.ErrNoRows) {
		return l, domain.ErrNoActiveVisualLanguage
	}
	return l, err
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}

// CreateVisualLanguage stores a Creator language at version 1.
func (r *PromptTemplateRepository) CreateVisualLanguage(ctx context.Context, l domain.VisualLanguage) (domain.VisualLanguage, error) {
	if err := l.Rules.Validate(); err != nil {
		return l, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return l, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO visual_languages (id, name, note, description, kit_key)
		VALUES (gen_random_uuid()::text, $1, $2, $3, $4) RETURNING id`,
		l.Name, l.Note, l.Description, l.KitKey).Scan(&id)
	if isUniqueViolation(err, "visual_languages_name_key") {
		return l, domain.ErrVisualLanguageNameTaken
	}
	if err != nil {
		return l, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO visual_language_versions (language_id, version, illustration_rules, backdrop_rules, scene_rules)
		VALUES ($1, 1, $2, $3, $4)`, id, l.Rules.Illustration, l.Rules.Backdrop, l.Rules.Scene); err != nil {
		return l, err
	}
	if err := tx.Commit(ctx); err != nil {
		return l, err
	}
	return r.GetVisualLanguage(ctx, id)
}

// AddVisualLanguageVersion stores the language's edited fields and rules as the next version.
func (r *PromptTemplateRepository) AddVisualLanguageVersion(ctx context.Context, l domain.VisualLanguage) (domain.VisualLanguage, error) {
	if err := l.Rules.Validate(); err != nil {
		return l, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return l, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE visual_languages SET name = $2, note = $3, description = $4 WHERE id = $1`,
		l.ID, l.Name, l.Note, l.Description)
	if isUniqueViolation(err, "visual_languages_name_key") {
		return l, domain.ErrVisualLanguageNameTaken
	}
	if err != nil {
		return l, err
	}
	if tag.RowsAffected() == 0 {
		return l, domain.ErrVisualLanguageNotFound
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO visual_language_versions (language_id, version, illustration_rules, backdrop_rules, scene_rules)
		SELECT $1, COALESCE(max(version), 0) + 1, $2, $3, $4 FROM visual_language_versions WHERE language_id = $1`,
		l.ID, l.Rules.Illustration, l.Rules.Backdrop, l.Rules.Scene); err != nil {
		return l, err
	}
	if err := tx.Commit(ctx); err != nil {
		return l, err
	}
	return r.GetVisualLanguage(ctx, l.ID)
}

// ActivateVisualLanguage makes id the only active language in one transaction.
func (r *PromptTemplateRepository) ActivateVisualLanguage(ctx context.Context, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE visual_languages SET is_active = false WHERE is_active AND id <> $1`, id); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE visual_languages SET is_active = true WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrVisualLanguageNotFound
	}
	return tx.Commit(ctx)
}

// DeleteVisualLanguage deletes a Creator language that is not active and that no project pinned.
func (r *PromptTemplateRepository) DeleteVisualLanguage(ctx context.Context, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var active bool
	if err := tx.QueryRow(ctx, `SELECT is_active FROM visual_languages WHERE id = $1 FOR UPDATE`, id).Scan(&active); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrVisualLanguageNotFound
		}
		return err
	}
	if active {
		return domain.ErrVisualLanguageActive
	}
	var pinned bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM project_authoring WHERE visual_language_id = $1)`, id).Scan(&pinned); err != nil {
		return err
	}
	if pinned {
		return domain.ErrVisualLanguageInUse
	}
	if _, err := tx.Exec(ctx, `DELETE FROM visual_languages WHERE id = $1 AND NOT is_system`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// PinnedLanguage returns the language version a project is pinned to.
func (r *PromptTemplateRepository) PinnedLanguage(ctx context.Context, projectID string) (domain.PinnedLanguage, error) {
	var p domain.PinnedLanguage
	err := r.pool.QueryRow(ctx, `
		SELECT a.visual_language_id, a.visual_language_version, l.kit_key, l.name,
		       v.illustration_rules, v.backdrop_rules, v.scene_rules
		FROM project_authoring a
		JOIN visual_languages l ON l.id = a.visual_language_id
		JOIN visual_language_versions v ON v.language_id = a.visual_language_id AND v.version = a.visual_language_version
		WHERE a.project_id = $1`, projectID).
		Scan(&p.ID, &p.Version, &p.KitKey, &p.Name, &p.Rules.Illustration, &p.Rules.Backdrop, &p.Rules.Scene)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, domain.ErrProjectNotPinned
	}
	return p, err
}

// ListLanguageSamples returns the recorded sample stills of a kit.
func (r *PromptTemplateRepository) ListLanguageSamples(ctx context.Context, kitKey string) ([]domain.LanguageSample, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT kit_key, kit_version, scene, frame, file FROM visual_language_samples
		WHERE kit_key = $1 ORDER BY scene, frame`, kitKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.LanguageSample
	for rows.Next() {
		var s domain.LanguageSample
		if err := rows.Scan(&s.KitKey, &s.KitVersion, &s.Scene, &s.Frame, &s.File); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// SaveLanguageSample records (or replaces) one sample still of a kit.
func (r *PromptTemplateRepository) SaveLanguageSample(ctx context.Context, s domain.LanguageSample) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO visual_language_samples (kit_key, scene, frame, kit_version, file)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (kit_key, scene, frame) DO UPDATE SET kit_version = EXCLUDED.kit_version, file = EXCLUDED.file, created_at = now()`,
		s.KitKey, s.Scene, s.Frame, s.KitVersion, s.File)
	return err
}

// SeedVisualLanguages makes sure the shipped language exists and its newest
// version carries the shipped rule texts (a changed text becomes a new
// version, so projects pinned to the old one keep it), and that one language
// is active.
func (r *PromptTemplateRepository) SeedVisualLanguages(ctx context.Context) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		INSERT INTO visual_languages (id, name, note, description, kit_key, is_system)
		VALUES ($1, 'Khoa học phát sáng', $2, $3, $4, true)
		ON CONFLICT (id) DO NOTHING`,
		domain.PhatsangID,
		"Tham khảo: phim khoa học phổ thông vẽ 2D, ánh sáng làm nhân vật chính của hình.",
		"Khối bo tròn không viền, ánh sáng kể chuyện: mọi khối có viền sáng về phía nguồn sáng của cảnh, vật phát sáng toả quầng. Màu theo thực tế của nơi chốn và bảng màu của video.",
		domain.PhatsangKitKey); err != nil {
		return err
	}
	rules := domain.PhatsangRules()
	var sameAsShipped bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM visual_language_versions
		  WHERE language_id = $1
		    AND version = (SELECT max(version) FROM visual_language_versions WHERE language_id = $1)
		    AND illustration_rules = $2 AND backdrop_rules = $3 AND scene_rules = $4)`,
		domain.PhatsangID, rules.Illustration, rules.Backdrop, rules.Scene).Scan(&sameAsShipped); err != nil {
		return err
	}
	if !sameAsShipped {
		if _, err := tx.Exec(ctx, `
			INSERT INTO visual_language_versions (language_id, version, illustration_rules, backdrop_rules, scene_rules)
			SELECT $1, COALESCE(max(version), 0) + 1, $2, $3, $4 FROM visual_language_versions WHERE language_id = $1`,
			domain.PhatsangID, rules.Illustration, rules.Backdrop, rules.Scene); err != nil {
			return fmt.Errorf("seed visual language rules: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE visual_languages SET is_active = true
		WHERE id = $1 AND NOT EXISTS (SELECT 1 FROM visual_languages WHERE is_active)`, domain.PhatsangID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
