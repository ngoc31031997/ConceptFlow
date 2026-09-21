package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"orchestrator/internal/domain"
)

// ErrPromptTemplateNotFound is returned by GetPromptTemplate when no row
// matches (role, language) — distinct from domain.ErrProjectNotFound since
// prompt templates are not part of the Project aggregate.
var ErrPromptTemplateNotFound = errors.New("prompt template not found")

// ErrPromptOverrideNotFound is returned when a role/language has no saved
// override — a normal state, not a failure: most roles run on the shipped
// wording (CR-027 FR84.4).
var ErrPromptOverrideNotFound = errors.New("prompt override not found")

// PromptTemplateRepository implements CR-025's prompt_templates CRUD.
type PromptTemplateRepository struct {
	pool *pgxpool.Pool
}

// NewPromptTemplateRepository constructs the repository over an already-open pool.
func NewPromptTemplateRepository(pool *pgxpool.Pool) *PromptTemplateRepository {
	return &PromptTemplateRepository{pool: pool}
}

// MigrateEditsToOverrides moves any hand-edited prompt into prompt_overrides
// before seeding overwrites prompt_templates (CR-027 FR84.8).
//
// MUST run before SeedPromptTemplates. Reversed, the seed has already
// replaced the very text this reads, and an edit the Creator made is gone
// with nothing left to compare against.
//
// A migrated edit is switched ON, so behaviour after the upgrade matches
// behaviour before it. Defaulting to the shipped wording "for a clean start"
// would silently change the prompts a Creator is working with — the exact
// class of surprise this CR exists to remove.
//
// Rows are compared by CONTENT, never by version. Measured on the live
// database 2026-09-21: story_architect sat at version 4 (vi) and 3 (en)
// while the binary shipped version 2, and the text matched byte for byte —
// because ResetPromptTemplate goes through Update, which always bumps the
// version. Trusting version here would manufacture two phantom overrides,
// switched on, identical to the shipped text, frozen there forever.
//
// Needs no "already migrated" flag: once seeding has run, prompt_templates
// equals the seed, so a second pass finds nothing to move.
// ShouldMigrateToOverride decides whether one stored row is a hand edit that
// must be preserved (CR-027 FR84.8). Exported so it can be tested without a
// live database — the rule is the whole risk of the migration, not the SQL
// around it.
//
// Compares TEXT, never Version. Measured on the live database 2026-09-21:
// story_architect sat at version 4 (vi) and 3 (en) against a binary shipping
// version 2, with text matching byte for byte — ResetPromptTemplate goes
// through Update, and Update always bumps the version. A version-based rule
// would fabricate two phantom overrides, switched on, identical to the
// shipped text, frozen there while the shipped wording moves on.
//
// A row whose role the binary no longer ships is kept as well: it cannot be
// compared to anything, and discarding wording nobody can recover is the
// worse mistake.
func ShouldMigrateToOverride(stored domain.PromptTemplate) bool {
	def, ok := domain.DefaultPromptTemplate(stored.Role, stored.Language)
	if !ok {
		return true
	}
	return stored.TemplateText != def.TemplateText
}

func (r *PromptTemplateRepository) MigrateEditsToOverrides(ctx context.Context) ([]domain.PromptTemplate, error) {
	stored, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	migrated := make([]domain.PromptTemplate, 0)
	for _, row := range stored {
		if !ShouldMigrateToOverride(row) {
			continue
		}
		if _, err := r.pool.Exec(ctx, `
			INSERT INTO prompt_overrides (role, language, template_text, is_active, updated_at)
			VALUES ($1, $2, $3, true, now())
			ON CONFLICT (role, language) DO NOTHING
		`, string(row.Role), row.Language, row.TemplateText); err != nil {
			return nil, err
		}
		migrated = append(migrated, row)
	}
	return migrated, nil
}

// SeedPromptTemplates writes the built-in default templates, overwriting what
// is there (CR-027 FR84.2).
//
// This used to be insert-if-absent, and its old docstring explained that
// version-aware overwriting had been tried and removed because it silently
// undid an editor's saved wording. That reasoning no longer applies: after
// CR-027 FR84 this table is not editable by anyone, so there is nothing of
// the Creator's here to undo. Their wording lives in prompt_overrides, which
// this function never touches.
//
// What overwriting buys is the end of a silent trap. Before, editing the
// wording in Go, rebuilding and restarting left the running database on the
// old text with no error, no warning and no way to notice short of reading
// the database — and the workaround was to remember to press Reset for each
// role and language.
func (r *PromptTemplateRepository) SeedPromptTemplates(ctx context.Context) error {
	for _, t := range domain.DefaultPromptTemplates() {
		if _, err := r.pool.Exec(ctx, `
			INSERT INTO prompt_templates (role, language, template_text, version, updated_at)
			VALUES ($1, $2, $3, $4, now())
			ON CONFLICT (role, language) DO UPDATE SET
			    template_text = EXCLUDED.template_text,
			    version = EXCLUDED.version,
			    updated_at = now()
		`, string(t.Role), t.Language, t.TemplateText, t.Version); err != nil {
			return err
		}
	}
	return nil
}

// Get returns one template, or ErrPromptTemplateNotFound.
func (r *PromptTemplateRepository) Get(ctx context.Context, role domain.PromptRole, language string) (domain.PromptTemplate, error) {
	var t domain.PromptTemplate
	var updatedAt time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT role, language, template_text, version, updated_at
		FROM prompt_templates WHERE role = $1 AND language = $2
	`, string(role), language).Scan(&t.Role, &t.Language, &t.TemplateText, &t.Version, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PromptTemplate{}, ErrPromptTemplateNotFound
	}
	if err != nil {
		return domain.PromptTemplate{}, err
	}
	t.UpdatedAt = updatedAt.Format(time.RFC3339)
	return t, nil
}

// List returns every template row (all roles/languages), for the admin screen.
func (r *PromptTemplateRepository) List(ctx context.Context) ([]domain.PromptTemplate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT role, language, template_text, version, updated_at
		FROM prompt_templates ORDER BY role, language
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.PromptTemplate, 0)
	for rows.Next() {
		var t domain.PromptTemplate
		var updatedAt time.Time
		if err := rows.Scan(&t.Role, &t.Language, &t.TemplateText, &t.Version, &updatedAt); err != nil {
			return nil, err
		}
		t.UpdatedAt = updatedAt.Format(time.RFC3339)
		out = append(out, t)
	}
	return out, rows.Err()
}

// Update overwrites a template's text and increments its version, or inserts
// it at version 1 if it does not exist yet (a role/language pair the seed
// step never created, or a language variant added later by an editor).
func (r *PromptTemplateRepository) Update(ctx context.Context, role domain.PromptRole, language, templateText string) (domain.PromptTemplate, error) {
	var t domain.PromptTemplate
	var updatedAt time.Time
	err := r.pool.QueryRow(ctx, `
		INSERT INTO prompt_templates (role, language, template_text, version, updated_at)
		VALUES ($1, $2, $3, 1, now())
		ON CONFLICT (role, language) DO UPDATE SET
		    template_text = EXCLUDED.template_text,
		    version = prompt_templates.version + 1,
		    updated_at = now()
		RETURNING role, language, template_text, version, updated_at
	`, string(role), language, templateText).Scan(&t.Role, &t.Language, &t.TemplateText, &t.Version, &updatedAt)
	if err != nil {
		return domain.PromptTemplate{}, err
	}
	t.UpdatedAt = updatedAt.Format(time.RFC3339)
	return t, nil
}

// --- CR-025 step 1: authoring story (Story Architect output) --------------

// SaveAuthoringStory upserts the pasted story outline for a project, plus the
// topic it was written from (CR-027 D0 — the topic is what {{topic}} renders
// to, and tab 1a is where the Creator types it).
//
// An empty topic leaves the stored one alone instead of clearing it. A caller
// that has no topic to offer — a browser still running pre-CR-027 JavaScript,
// or any later save that only means to replace the outline — must not wipe a
// topic the Creator already gave us, since every later pipeline step renders
// its prompt from it.
func (r *PromptTemplateRepository) SaveAuthoringStory(ctx context.Context, projectID, content, topic string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO project_authoring (project_id, story_content, topic, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (project_id) DO UPDATE SET
		    story_content = EXCLUDED.story_content,
		    topic = CASE WHEN EXCLUDED.topic = '' THEN project_authoring.topic
		                 ELSE EXCLUDED.topic END,
		    updated_at = now()
	`, projectID, content, topic)
	return err
}

// GetAuthoringTopic returns the saved topic, or "" if none was saved yet
// (every project created before CR-027 D0).
func (r *PromptTemplateRepository) GetAuthoringTopic(ctx context.Context, projectID string) (string, error) {
	var topic string
	err := r.pool.QueryRow(ctx, `
		SELECT topic FROM project_authoring WHERE project_id = $1
	`, projectID).Scan(&topic)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return topic, err
}

// GetAuthoringStory returns the saved story outline, or "" if none was saved yet.
func (r *PromptTemplateRepository) GetAuthoringStory(ctx context.Context, projectID string) (string, error) {
	var content string
	err := r.pool.QueryRow(ctx, `
		SELECT story_content FROM project_authoring WHERE project_id = $1
	`, projectID).Scan(&content)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return content, err
}

// --- CR-025 step 2: authoring storyboard (Visual Director output) ---------

// SaveAuthoringStoryboard upserts the pasted storyboard for a project. Same
// insert-if-absent-row/update-in-place shape as SaveAuthoringStory, sharing
// the same project_authoring row (story and storyboard are both keyed by
// project_id, so a project row created by only one of the two saves still
// gets the other column filled in on ON CONFLICT rather than erroring).
func (r *PromptTemplateRepository) SaveAuthoringStoryboard(ctx context.Context, projectID, content string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO project_authoring (project_id, storyboard_content, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (project_id) DO UPDATE SET
		    storyboard_content = EXCLUDED.storyboard_content, updated_at = now()
	`, projectID, content)
	return err
}

// GetAuthoringStoryboard returns the saved storyboard, or "" if none was saved yet.
func (r *PromptTemplateRepository) GetAuthoringStoryboard(ctx context.Context, projectID string) (string, error) {
	var content string
	err := r.pool.QueryRow(ctx, `
		SELECT storyboard_content FROM project_authoring WHERE project_id = $1
	`, projectID).Scan(&content)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return content, err
}

// --- CR-025 step 3: authoring code (Manim Engineer output) ----------------

// SaveAuthoringCode upserts the pasted Manim code for a project. Same
// insert-if-absent-row/update-in-place shape as SaveAuthoringStory/
// SaveAuthoringStoryboard, sharing the same project_authoring row.
func (r *PromptTemplateRepository) SaveAuthoringCode(ctx context.Context, projectID, content string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO project_authoring (project_id, code_content, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (project_id) DO UPDATE SET
		    code_content = EXCLUDED.code_content, updated_at = now()
	`, projectID, content)
	return err
}

// GetAuthoringCode returns the saved Manim code, or "" if none was saved yet.
func (r *PromptTemplateRepository) GetAuthoringCode(ctx context.Context, projectID string) (string, error) {
	var content string
	err := r.pool.QueryRow(ctx, `
		SELECT code_content FROM project_authoring WHERE project_id = $1
	`, projectID).Scan(&content)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return content, err
}

// --- CR-025 step 4: authoring review (Script Reviewer verdict) ------------

// SaveAuthoringReview upserts the pasted reviewer verdict for a project. Same
// shape as the other three authoring saves above.
func (r *PromptTemplateRepository) SaveAuthoringReview(ctx context.Context, projectID, content string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO project_authoring (project_id, review_content, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (project_id) DO UPDATE SET
		    review_content = EXCLUDED.review_content, updated_at = now()
	`, projectID, content)
	return err
}

// GetAuthoringReview returns the saved reviewer verdict, or "" if none was saved yet.
func (r *PromptTemplateRepository) GetAuthoringReview(ctx context.Context, projectID string) (string, error) {
	var content string
	err := r.pool.QueryRow(ctx, `
		SELECT review_content FROM project_authoring WHERE project_id = $1
	`, projectID).Scan(&content)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return content, err
}

// --- CR-027 FR84: the Creator's own wording -------------------------------

// GetEffective returns the template the pipeline should actually use: the
// Creator's override when one is switched on, otherwise the shipped text
// (FR84.4).
//
// One query rather than two round trips, so "is there an active override"
// and "what is the shipped text" can never be answered from two different
// moments in time.
func (r *PromptTemplateRepository) GetEffective(ctx context.Context, role domain.PromptRole, language string) (domain.EffectivePromptTemplate, error) {
	var out domain.EffectivePromptTemplate
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(o.template_text, t.template_text),
		       (o.template_text IS NOT NULL),
		       t.version
		FROM prompt_templates t
		LEFT JOIN prompt_overrides o
		       ON o.role = t.role AND o.language = t.language AND o.is_active
		WHERE t.role = $1 AND t.language = $2
	`, string(role), language).Scan(&out.TemplateText, &out.FromOverride, &out.SeedVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.EffectivePromptTemplate{}, ErrPromptTemplateNotFound
	}
	if err != nil {
		return domain.EffectivePromptTemplate{}, err
	}
	out.Role = role
	out.Language = language
	return out, nil
}

// GetOverride returns the Creator's saved wording for one role/language.
func (r *PromptTemplateRepository) GetOverride(ctx context.Context, role domain.PromptRole, language string) (domain.PromptOverride, error) {
	var o domain.PromptOverride
	var updatedAt time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT role, language, template_text, is_active, updated_at
		FROM prompt_overrides WHERE role = $1 AND language = $2
	`, string(role), language).Scan(&o.Role, &o.Language, &o.TemplateText, &o.IsActive, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PromptOverride{}, ErrPromptOverrideNotFound
	}
	if err != nil {
		return domain.PromptOverride{}, err
	}
	o.UpdatedAt = updatedAt.Format(time.RFC3339)
	return o, nil
}

// ListOverrides returns every saved override, for the admin screen.
func (r *PromptTemplateRepository) ListOverrides(ctx context.Context) ([]domain.PromptOverride, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT role, language, template_text, is_active, updated_at
		FROM prompt_overrides ORDER BY role, language
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.PromptOverride, 0)
	for rows.Next() {
		var o domain.PromptOverride
		var updatedAt time.Time
		if err := rows.Scan(&o.Role, &o.Language, &o.TemplateText, &o.IsActive, &updatedAt); err != nil {
			return nil, err
		}
		o.UpdatedAt = updatedAt.Format(time.RFC3339)
		out = append(out, o)
	}
	return out, rows.Err()
}

// SaveOverride writes the Creator's wording, leaving is_active alone on an
// existing row: editing the text of an override that is currently off must
// not silently switch it on. Switching is its own deliberate action.
func (r *PromptTemplateRepository) SaveOverride(ctx context.Context, role domain.PromptRole, language, templateText string) (domain.PromptOverride, error) {
	var o domain.PromptOverride
	var updatedAt time.Time
	err := r.pool.QueryRow(ctx, `
		INSERT INTO prompt_overrides (role, language, template_text, is_active, updated_at)
		VALUES ($1, $2, $3, true, now())
		ON CONFLICT (role, language) DO UPDATE SET
		    template_text = EXCLUDED.template_text,
		    updated_at = now()
		RETURNING role, language, template_text, is_active, updated_at
	`, string(role), language, templateText).Scan(&o.Role, &o.Language, &o.TemplateText, &o.IsActive, &updatedAt)
	if err != nil {
		return domain.PromptOverride{}, err
	}
	o.UpdatedAt = updatedAt.Format(time.RFC3339)
	return o, nil
}

// SetOverrideActive switches an existing override on or off (FR84.5).
//
// Switching off is the non-destructive replacement for the old reset
// endpoint: the wording stays, and switching back on restores it. Reset threw
// the Creator's text away and could not be undone.
func (r *PromptTemplateRepository) SetOverrideActive(ctx context.Context, role domain.PromptRole, language string, active bool) (domain.PromptOverride, error) {
	var o domain.PromptOverride
	var updatedAt time.Time
	err := r.pool.QueryRow(ctx, `
		UPDATE prompt_overrides SET is_active = $3, updated_at = now()
		WHERE role = $1 AND language = $2
		RETURNING role, language, template_text, is_active, updated_at
	`, string(role), language, active).Scan(&o.Role, &o.Language, &o.TemplateText, &o.IsActive, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PromptOverride{}, ErrPromptOverrideNotFound
	}
	if err != nil {
		return domain.PromptOverride{}, err
	}
	o.UpdatedAt = updatedAt.Format(time.RFC3339)
	return o, nil
}

// DeleteOverride removes the Creator's wording entirely, leaving the shipped
// text in charge.
func (r *PromptTemplateRepository) DeleteOverride(ctx context.Context, role domain.PromptRole, language string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM prompt_overrides WHERE role = $1 AND language = $2
	`, string(role), language)
	return err
}
