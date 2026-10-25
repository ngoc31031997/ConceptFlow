package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"authoring/internal/domain"
)

// The gate of the research step lives in project_research_gate, one row per
// project: whether the Creator skipped the step, the number the next question
// takes and the approval of the fact table. The step reads and writes it as
// the research review unit, so the unit store routes that kind here.

func (r *PromptTemplateRepository) getResearchGateUnit(ctx context.Context, projectID, key string) (domain.ReviewUnit, error) {
	if key != domain.ResearchUnitKey {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	u := domain.ReviewUnit{ProjectID: projectID, Kind: domain.UnitKindResearch, Key: key}
	var g domain.ResearchGate
	var status string
	err := r.pool.QueryRow(ctx, `
		SELECT skipped, next_question, status, creator_fields, redo_count, updated_at
		FROM project_research_gate WHERE project_id = $1`, projectID).Scan(
		&g.Skipped, &g.NextQuestion, &status, &u.CreatorFields, &u.RedoCount, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	if u.Content, err = json.Marshal(g); err != nil {
		return domain.ReviewUnit{}, err
	}
	u.Status = domain.UnitStatus(status)
	if u.CreatorFields == nil {
		u.CreatorFields = []string{}
	}
	return u, nil
}

func (r *PromptTemplateRepository) saveResearchGateUnit(ctx context.Context, unit domain.ReviewUnit) (domain.ReviewUnit, error) {
	var g domain.ResearchGate
	if len(unit.Content) > 0 {
		if err := json.Unmarshal(unit.Content, &g); err != nil {
			return domain.ReviewUnit{}, fmt.Errorf("read research gate: %w", err)
		}
	}
	fields := unit.CreatorFields
	if fields == nil {
		fields = []string{}
	}
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO project_research_gate (project_id, skipped, next_question, status, creator_fields, redo_count, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, now())
		ON CONFLICT (project_id) DO UPDATE SET
		    skipped = EXCLUDED.skipped, next_question = EXCLUDED.next_question, status = EXCLUDED.status,
		    creator_fields = EXCLUDED.creator_fields, redo_count = EXCLUDED.redo_count, updated_at = now()`,
		unit.ProjectID, g.Skipped, g.NextQuestion, string(unit.Status), fields, unit.RedoCount); err != nil {
		return domain.ReviewUnit{}, err
	}
	return r.getResearchGateUnit(ctx, unit.ProjectID, domain.ResearchUnitKey)
}
