package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"authoring/internal/domain"
)

// GetProjectIdea returns a project's idea, or domain.ErrIdeaNotFound.
func (r *PromptTemplateRepository) GetProjectIdea(ctx context.Context, projectID string) (domain.StoredIdea, error) {
	var s domain.StoredIdea
	var points, creator []byte
	var status string
	err := r.pool.QueryRow(ctx, `
		SELECT project_id, central_question, curiosity_points, scope, creator_fields,
		       status, redo_count, language, input_hash, approved_at, updated_at
		FROM project_idea WHERE project_id = $1`, projectID).Scan(
		&s.ProjectID, &s.Idea.CentralQuestion, &points, &s.Idea.Scope, &creator,
		&status, &s.RedoCount, &s.Language, &s.InputHash, &s.ApprovedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.StoredIdea{}, domain.ErrIdeaNotFound
	}
	if err != nil {
		return domain.StoredIdea{}, err
	}
	if err := json.Unmarshal(points, &s.Idea.CuriosityPoints); err != nil {
		return domain.StoredIdea{}, fmt.Errorf("read curiosity points: %w", err)
	}
	if err := json.Unmarshal(creator, &s.CreatorFields); err != nil {
		return domain.StoredIdea{}, fmt.Errorf("read creator fields: %w", err)
	}
	s.Status = domain.IdeaStatus(status)
	return s, nil
}

// SaveProjectIdea inserts or replaces a project's idea.
func (r *PromptTemplateRepository) SaveProjectIdea(ctx context.Context, s domain.StoredIdea) error {
	if s.Idea.CuriosityPoints == nil {
		s.Idea.CuriosityPoints = []string{}
	}
	if s.CreatorFields == nil {
		s.CreatorFields = []string{}
	}
	points, err := json.Marshal(s.Idea.CuriosityPoints)
	if err != nil {
		return err
	}
	creator, err := json.Marshal(s.CreatorFields)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO project_idea (project_id, central_question, curiosity_points, scope, creator_fields,
		                          status, redo_count, language, input_hash, approved_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (project_id) DO UPDATE SET
		    central_question = EXCLUDED.central_question,
		    curiosity_points = EXCLUDED.curiosity_points, scope = EXCLUDED.scope,
		    creator_fields = EXCLUDED.creator_fields, status = EXCLUDED.status, redo_count = EXCLUDED.redo_count,
		    language = EXCLUDED.language, input_hash = EXCLUDED.input_hash,
		    approved_at = EXCLUDED.approved_at, updated_at = EXCLUDED.updated_at`,
		s.ProjectID, s.Idea.CentralQuestion, points, s.Idea.Scope, creator,
		string(s.Status), s.RedoCount, s.Language, s.InputHash, s.ApprovedAt, s.UpdatedAt)
	return err
}

// GetIdeaReview returns the latest review of a project's idea, or
// domain.ErrIdeaNotFound.
func (r *PromptTemplateRepository) GetIdeaReview(ctx context.Context, projectID string) (domain.IdeaReview, error) {
	var raw []byte
	err := r.pool.QueryRow(ctx, `SELECT review FROM project_idea_review WHERE project_id = $1`, projectID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.IdeaReview{}, domain.ErrIdeaNotFound
	}
	if err != nil {
		return domain.IdeaReview{}, err
	}
	var review domain.IdeaReview
	if err := json.Unmarshal(raw, &review); err != nil {
		return domain.IdeaReview{}, fmt.Errorf("read idea review: %w", err)
	}
	return review, nil
}

// SaveIdeaReview replaces the review of a project's idea.
func (r *PromptTemplateRepository) SaveIdeaReview(ctx context.Context, projectID string, review domain.IdeaReview) error {
	raw, err := json.Marshal(review)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO project_idea_review (project_id, review, idea_hash, created_at) VALUES ($1, $2, $3, $4)
		ON CONFLICT (project_id) DO UPDATE SET review = EXCLUDED.review, idea_hash = EXCLUDED.idea_hash, created_at = EXCLUDED.created_at`,
		projectID, raw, review.IdeaHash, review.CreatedAt)
	return err
}
