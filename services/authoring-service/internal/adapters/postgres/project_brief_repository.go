package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"authoring/internal/domain"
)

// GetProjectBrief returns a project's brief, or domain.ErrBriefNotFound.
func (r *PromptTemplateRepository) GetProjectBrief(ctx context.Context, projectID string) (domain.StoredBrief, error) {
	var s domain.StoredBrief
	var objectives, avoid, creator []byte
	var status string
	err := r.pool.QueryRow(ctx, `
		SELECT project_id, audience, duration_seconds, tone, learning_objectives, avoid, creator_fields,
		       status, redo_count, language, idea_hash, approved_at, updated_at
		FROM project_brief WHERE project_id = $1`, projectID).Scan(
		&s.ProjectID, &s.Brief.Audience, &s.Brief.DurationSeconds, &s.Brief.Tone, &objectives, &avoid, &creator,
		&status, &s.RedoCount, &s.Language, &s.IdeaHash, &s.ApprovedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.StoredBrief{}, domain.ErrBriefNotFound
	}
	if err != nil {
		return domain.StoredBrief{}, err
	}
	if err := json.Unmarshal(objectives, &s.Brief.LearningObjectives); err != nil {
		return domain.StoredBrief{}, fmt.Errorf("read learning objectives: %w", err)
	}
	if err := json.Unmarshal(avoid, &s.Brief.Avoid); err != nil {
		return domain.StoredBrief{}, fmt.Errorf("read avoid list: %w", err)
	}
	if err := json.Unmarshal(creator, &s.CreatorFields); err != nil {
		return domain.StoredBrief{}, fmt.Errorf("read creator fields: %w", err)
	}
	s.Status = domain.BriefStatus(status)
	return s, nil
}

// SaveProjectBrief inserts or replaces a project's brief.
func (r *PromptTemplateRepository) SaveProjectBrief(ctx context.Context, s domain.StoredBrief) error {
	if s.Brief.LearningObjectives == nil {
		s.Brief.LearningObjectives = []string{}
	}
	if s.Brief.Avoid == nil {
		s.Brief.Avoid = []string{}
	}
	if s.CreatorFields == nil {
		s.CreatorFields = []string{}
	}
	objectives, err := json.Marshal(s.Brief.LearningObjectives)
	if err != nil {
		return err
	}
	avoid, err := json.Marshal(s.Brief.Avoid)
	if err != nil {
		return err
	}
	creator, err := json.Marshal(s.CreatorFields)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO project_brief (project_id, audience, duration_seconds, tone, learning_objectives, avoid,
		                           creator_fields, status, redo_count, language, idea_hash, approved_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (project_id) DO UPDATE SET
		    audience = EXCLUDED.audience, duration_seconds = EXCLUDED.duration_seconds, tone = EXCLUDED.tone,
		    learning_objectives = EXCLUDED.learning_objectives, avoid = EXCLUDED.avoid,
		    creator_fields = EXCLUDED.creator_fields, status = EXCLUDED.status, redo_count = EXCLUDED.redo_count,
		    language = EXCLUDED.language, idea_hash = EXCLUDED.idea_hash,
		    approved_at = EXCLUDED.approved_at, updated_at = EXCLUDED.updated_at`,
		s.ProjectID, s.Brief.Audience, s.Brief.DurationSeconds, s.Brief.Tone, objectives, avoid,
		creator, string(s.Status), s.RedoCount, s.Language, s.IdeaHash, s.ApprovedAt, s.UpdatedAt)
	return err
}

// GetBriefReview returns the latest review of a project's brief, or
// domain.ErrBriefNotFound.
func (r *PromptTemplateRepository) GetBriefReview(ctx context.Context, projectID string) (domain.BriefReview, error) {
	var raw []byte
	err := r.pool.QueryRow(ctx, `SELECT review FROM project_brief_review WHERE project_id = $1`, projectID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.BriefReview{}, domain.ErrBriefNotFound
	}
	if err != nil {
		return domain.BriefReview{}, err
	}
	var review domain.BriefReview
	if err := json.Unmarshal(raw, &review); err != nil {
		return domain.BriefReview{}, fmt.Errorf("read brief review: %w", err)
	}
	return review, nil
}

// SaveBriefReview replaces the review of a project's brief.
func (r *PromptTemplateRepository) SaveBriefReview(ctx context.Context, projectID string, review domain.BriefReview) error {
	raw, err := json.Marshal(review)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO project_brief_review (project_id, review, brief_hash, created_at) VALUES ($1, $2, $3, $4)
		ON CONFLICT (project_id) DO UPDATE SET review = EXCLUDED.review, brief_hash = EXCLUDED.brief_hash, created_at = EXCLUDED.created_at`,
		projectID, raw, review.BriefHash, review.CreatedAt)
	return err
}
