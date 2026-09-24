package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"orchestrator/internal/application"
	"orchestrator/internal/domain"
)

// PromptTemplateRepository holds the authoring-pipeline artefacts per project
// (story / storyboard / code / topic / mode / models). The prompt library itself
// is in prompt_repository.go.
type PromptTemplateRepository struct {
	pool *pgxpool.Pool
}

// NewPromptTemplateRepository constructs the repository over an already-open pool.
func NewPromptTemplateRepository(pool *pgxpool.Pool) *PromptTemplateRepository {
	return &PromptTemplateRepository{pool: pool}
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

// SaveAuthoringTopic upserts just the topic (CR-028 FR83.1/FR83.2) — unlike
// SaveAuthoringStory, content is not required here: this is called at step 1
// of the wizard, before any outline exists.
func (r *PromptTemplateRepository) SaveAuthoringTopic(ctx context.Context, projectID, topic string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO project_authoring (project_id, topic, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (project_id) DO UPDATE SET
		    topic = EXCLUDED.topic,
		    updated_at = now()
	`, projectID, topic)
	return err
}

// FindSimilarTopics backs CR-028 FR85 — other projects, in the same
// content_language, whose saved topic normalizes to the same string as
// normalizedTopic. excludeProjectID keeps a project from "colliding" with
// its own topic when the Creator re-saves it unchanged. An empty
// normalizedTopic never matches anything (an unset topic is not a
// collision).
func (r *PromptTemplateRepository) FindSimilarTopics(ctx context.Context, language domain.ContentLanguage, normalizedTopic, excludeProjectID string) ([]application.SimilarProject, error) {
	if normalizedTopic == "" {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT p.project_id, a.topic, p.status, p.created_at
		FROM project_authoring a
		JOIN projects p ON p.project_id = a.project_id
		WHERE p.voice_language = $1
		  AND p.project_id != $2
		  AND lower(regexp_replace(trim(a.topic), '\s+', ' ', 'g')) = $3
		ORDER BY p.created_at DESC
		LIMIT 10
	`, string(language), excludeProjectID, normalizedTopic)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []application.SimilarProject
	for rows.Next() {
		var s application.SimilarProject
		var status string
		if err := rows.Scan(&s.ProjectID, &s.Topic, &status, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.Status = domain.ProjectStatus(status)
		out = append(out, s)
	}
	return out, rows.Err()
}

// ClearAuthoringSteps empties the saved output of the given steps. Only the
// three chained outputs can be named, so the column list below is fixed.
func (r *PromptTemplateRepository) ClearAuthoringSteps(ctx context.Context, projectID string, steps ...application.AuthoringStep) error {
	columns := map[application.AuthoringStep]string{
		application.AuthoringStepStory:      "story_content",
		application.AuthoringStepStoryboard: "storyboard_content",
		application.AuthoringStepCode:       "code_content",
	}
	var sets []string
	for _, step := range steps {
		column, ok := columns[step]
		if !ok {
			return fmt.Errorf("unknown authoring step %q", step)
		}
		sets = append(sets, column+" = ''")
	}
	if len(sets) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx,
		"UPDATE project_authoring SET "+strings.Join(sets, ", ")+", updated_at = now() WHERE project_id = $1",
		projectID)
	return err
}

// GetStatus is the narrow read CR-028 FR84.2's authoring lock needs — just
// enough to decide draft-vs-locked without paying for the full 49-column
// Project scan Get does.
func (r *PromptTemplateRepository) GetStatus(ctx context.Context, projectID string) (domain.ProjectStatus, error) {
	var status string
	err := r.pool.QueryRow(ctx, `SELECT status FROM projects WHERE project_id = $1`, projectID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrProjectNotFound
	}
	if err != nil {
		return "", err
	}
	return domain.ProjectStatus(status), nil
}

// GetStatusAndLanguage is GetStatus plus content_language, for FR83.2's
// re-scan of the collision list (comparison stays scoped to the project's
// own language).
func (r *PromptTemplateRepository) GetStatusAndLanguage(ctx context.Context, projectID string) (domain.ProjectStatus, domain.ContentLanguage, error) {
	var status, language string
	err := r.pool.QueryRow(ctx, `SELECT status, voice_language FROM projects WHERE project_id = $1`, projectID).Scan(&status, &language)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", domain.ErrProjectNotFound
	}
	if err != nil {
		return "", "", err
	}
	return domain.ProjectStatus(status), domain.ContentLanguage(language), nil
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

// --- CR-027 FR79: how the Creator works step 1 ----------------------------

// SaveAuthoringMode upserts the step-1 working mode ("manual" or "ai").
//
// Upsert on project_id alone, like SaveAuthoringTopic: the mode can be chosen
// on tab 1a before any outline exists, so it cannot wait for a content row.
func (r *PromptTemplateRepository) SaveAuthoringMode(ctx context.Context, projectID, mode string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO project_authoring (project_id, authoring_mode, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (project_id) DO UPDATE SET
		    authoring_mode = EXCLUDED.authoring_mode, updated_at = now()
	`, projectID, mode)
	return err
}

// GetAuthoringMode returns the saved mode, or "" when this project has no
// authoring row yet. "" is not an error and not a third mode — the caller
// treats it as the default, which is "manual".
func (r *PromptTemplateRepository) GetAuthoringMode(ctx context.Context, projectID string) (string, error) {
	var mode string
	err := r.pool.QueryRow(ctx, `
		SELECT authoring_mode FROM project_authoring WHERE project_id = $1
	`, projectID).Scan(&mode)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return mode, err
}

// SaveAuthoringModels upserts the per-step Hive model choice (model-per-step
// follow-up to CR-027) — all three tabs in one write, mirroring how the GUI
// saves them (one picker, at step 1, for all of 1a/1b/1c at once).
//
// Upsert on project_id alone, like SaveAuthoringMode: the choice can be made
// on tab 1a before any outline exists.
func (r *PromptTemplateRepository) SaveAuthoringModels(ctx context.Context, projectID string, models domain.AuthoringStepModels) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO project_authoring (project_id, story_model, storyboard_model, code_model, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (project_id) DO UPDATE SET
		    story_model = EXCLUDED.story_model,
		    storyboard_model = EXCLUDED.storyboard_model,
		    code_model = EXCLUDED.code_model,
		    updated_at = now()
	`, projectID, models.Story, models.Storyboard, models.Code)
	return err
}

// GetAuthoringModels returns the saved per-step model choice, or the zero
// value (every step "" — the server default) when this project has no
// authoring row yet or predates this column.
func (r *PromptTemplateRepository) GetAuthoringModels(ctx context.Context, projectID string) (domain.AuthoringStepModels, error) {
	var models domain.AuthoringStepModels
	err := r.pool.QueryRow(ctx, `
		SELECT story_model, storyboard_model, code_model FROM project_authoring WHERE project_id = $1
	`, projectID).Scan(&models.Story, &models.Storyboard, &models.Code)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AuthoringStepModels{}, nil
	}
	return models, err
}
