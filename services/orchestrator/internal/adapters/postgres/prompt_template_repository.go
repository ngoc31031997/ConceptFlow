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

// PromptTemplateRepository implements CR-025's prompt_templates CRUD.
type PromptTemplateRepository struct {
	pool *pgxpool.Pool
}

// NewPromptTemplateRepository constructs the repository over an already-open pool.
func NewPromptTemplateRepository(pool *pgxpool.Pool) *PromptTemplateRepository {
	return &PromptTemplateRepository{pool: pool}
}

// SeedPromptTemplates writes the built-in default templates if a (role,
// language) row is not already there — insert-if-absent, same reasoning as
// SeedVideoFormats: once an editor has changed the wording, a restart must
// not quietly restore the shipped copy underneath them.
func (r *PromptTemplateRepository) SeedPromptTemplates(ctx context.Context) error {
	for _, t := range domain.DefaultPromptTemplates() {
		if _, err := r.pool.Exec(ctx, `
			INSERT INTO prompt_templates (role, language, template_text, version)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (role, language) DO NOTHING
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

// SaveAuthoringStory upserts the pasted story outline for a project.
func (r *PromptTemplateRepository) SaveAuthoringStory(ctx context.Context, projectID, content string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO project_authoring (project_id, story_content, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (project_id) DO UPDATE SET
		    story_content = EXCLUDED.story_content, updated_at = now()
	`, projectID, content)
	return err
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
