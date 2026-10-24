package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"authoring/internal/domain"
)

// GetProjectStory returns a project's story, or domain.ErrStoryNotFound.
func (r *PromptTemplateRepository) GetProjectStory(ctx context.Context, projectID string) (domain.StoredStory, error) {
	var s domain.StoredStory
	var arc, creator []byte
	var status string
	err := r.pool.QueryRow(ctx, `
		SELECT project_id, central_question, core_answer, narrative_arc, creator_fields,
		       brief_hash, research_hash, status, created_at, updated_at
		FROM project_story WHERE project_id = $1`, projectID).Scan(
		&s.ProjectID, &s.Story.CentralQuestion, &s.Story.CoreAnswer, &arc, &creator,
		&s.BriefHash, &s.ResearchHash, &status, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.StoredStory{}, domain.ErrStoryNotFound
	}
	if err != nil {
		return domain.StoredStory{}, err
	}
	if err := json.Unmarshal(arc, &s.Story.NarrativeArc); err != nil {
		return domain.StoredStory{}, fmt.Errorf("read narrative arc: %w", err)
	}
	if err := json.Unmarshal(creator, &s.CreatorFields); err != nil {
		return domain.StoredStory{}, fmt.Errorf("read creator fields: %w", err)
	}
	if s.Story.NarrativeArc == nil {
		s.Story.NarrativeArc = []domain.ArcBeat{}
	}
	if s.CreatorFields == nil {
		s.CreatorFields = []string{}
	}
	s.Status = domain.StoryStatus(status)
	return s, nil
}

// SaveProjectStory inserts or replaces a project's story.
func (r *PromptTemplateRepository) SaveProjectStory(ctx context.Context, s domain.StoredStory) error {
	if s.Story.NarrativeArc == nil {
		s.Story.NarrativeArc = []domain.ArcBeat{}
	}
	for i := range s.Story.NarrativeArc {
		if s.Story.NarrativeArc[i].Facts == nil {
			s.Story.NarrativeArc[i].Facts = []string{}
		}
	}
	if s.CreatorFields == nil {
		s.CreatorFields = []string{}
	}
	arc, err := json.Marshal(s.Story.NarrativeArc)
	if err != nil {
		return err
	}
	creator, err := json.Marshal(s.CreatorFields)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO project_story (project_id, central_question, core_answer, narrative_arc, creator_fields,
		                           brief_hash, research_hash, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (project_id) DO UPDATE SET
		    central_question = EXCLUDED.central_question, core_answer = EXCLUDED.core_answer,
		    narrative_arc = EXCLUDED.narrative_arc, creator_fields = EXCLUDED.creator_fields,
		    brief_hash = EXCLUDED.brief_hash, research_hash = EXCLUDED.research_hash,
		    status = EXCLUDED.status, updated_at = EXCLUDED.updated_at`,
		s.ProjectID, s.Story.CentralQuestion, s.Story.CoreAnswer, arc, creator,
		s.BriefHash, s.ResearchHash, string(s.Status), s.CreatedAt, s.UpdatedAt)
	return err
}
