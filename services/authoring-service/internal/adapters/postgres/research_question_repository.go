package postgres

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"authoring/internal/domain"
)

// The research questions live in project_research_question, one row per
// question code, apart from the facts in project_research: a question has none
// of a fact's columns, and a fact listing must never return a question. The
// step reads and writes a question as a research_question review unit, so the
// unit store routes that kind here.

func scanResearchQuestionUnit(row pgx.Row, projectID string) (domain.ReviewUnit, error) {
	u := domain.ReviewUnit{ProjectID: projectID, Kind: domain.UnitKindResearchQuestion}
	var q domain.ResearchQuestion
	var status string
	err := row.Scan(&u.Key, &q.Text, &q.BriefHash, &status, &u.CreatorFields, &u.RedoCount, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	if u.Content, err = json.Marshal(q); err != nil {
		return domain.ReviewUnit{}, err
	}
	u.Status = domain.UnitStatus(status)
	if u.CreatorFields == nil {
		u.CreatorFields = []string{}
	}
	return u, nil
}

const researchQuestionColumns = `question_id, text, brief_hash, status, creator_fields, redo_count, updated_at`

func (r *PromptTemplateRepository) getResearchQuestionUnit(ctx context.Context, projectID, key string) (domain.ReviewUnit, error) {
	return scanResearchQuestionUnit(r.pool.QueryRow(ctx,
		`SELECT `+researchQuestionColumns+` FROM project_research_question WHERE project_id = $1 AND question_id = $2`,
		projectID, key), projectID)
}

// listResearchQuestionUnits returns the questions in code order (Q2 before Q10).
func (r *PromptTemplateRepository) listResearchQuestionUnits(ctx context.Context, projectID string) ([]domain.ReviewUnit, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+researchQuestionColumns+` FROM project_research_question WHERE project_id = $1
		 ORDER BY length(question_id), question_id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	units := []domain.ReviewUnit{}
	for rows.Next() {
		u, err := scanResearchQuestionUnit(rows, projectID)
		if err != nil {
			return nil, err
		}
		units = append(units, u)
	}
	return units, rows.Err()
}

func (r *PromptTemplateRepository) saveResearchQuestionUnit(ctx context.Context, unit domain.ReviewUnit) (domain.ReviewUnit, error) {
	var q domain.ResearchQuestion
	if len(unit.Content) > 0 {
		if err := json.Unmarshal(unit.Content, &q); err != nil {
			return domain.ReviewUnit{}, fmt.Errorf("read research question: %w", err)
		}
	}
	fields := unit.CreatorFields
	if fields == nil {
		fields = []string{}
	}
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO project_research_question (project_id, question_id, text, brief_hash, status, creator_fields, redo_count, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now())
		ON CONFLICT (project_id, question_id) DO UPDATE SET
		    text = EXCLUDED.text, brief_hash = EXCLUDED.brief_hash, status = EXCLUDED.status,
		    creator_fields = EXCLUDED.creator_fields, redo_count = EXCLUDED.redo_count, updated_at = now()`,
		unit.ProjectID, unit.Key, q.Text, q.BriefHash, string(unit.Status), fields, unit.RedoCount); err != nil {
		return domain.ReviewUnit{}, err
	}
	return r.getResearchQuestionUnit(ctx, unit.ProjectID, unit.Key)
}

func (r *PromptTemplateRepository) deleteResearchQuestionUnit(ctx context.Context, projectID, key string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM project_research_question WHERE project_id = $1 AND question_id = $2`, projectID, key)
	return err
}

// researchQuestionSHA hashes a question's stored content; "" when it does not
// exist.
func (r *PromptTemplateRepository) researchQuestionSHA(ctx context.Context, projectID, key string) (string, error) {
	u, err := r.getResearchQuestionUnit(ctx, projectID, key)
	if errors.Is(err, domain.ErrUnitNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	sum := md5.Sum(u.Content)
	return hex.EncodeToString(sum[:]), nil
}
