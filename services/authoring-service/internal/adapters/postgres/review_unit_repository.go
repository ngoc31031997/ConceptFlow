package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"authoring/internal/domain"
)

// approvedBriefSHA is the md5 of the project's approved brief ($1 is the
// project id), or the empty string when no brief is approved. jsonb prints its keys in a
// fixed order, so equal briefs hash equally.
const approvedBriefSHA = `COALESCE((
    SELECT md5(ru.content::text) FROM review_units ru
    WHERE ru.project_id = $1 AND ru.kind = 'brief' AND ru.unit_key = 'main' AND ru.status = 'approved'
), '')`

// storyContentSHA hashes the content of the story of project $1: the same hash
// a unit records as input_sha when it names the story as its input (see
// domain.StoryInputKind), so that a unit written from a story that has changed
// since reads as stale.
const storyContentSHA = `(SELECT md5(ps.central_question || E'\n' || ps.core_answer || E'\n' || ps.narrative_arc::text)
    FROM project_story ps WHERE ps.project_id = $1)`

// reviewUnitColumns lists a unit's columns for a query that aliases the table
// as ru. The last column is stale: the unit was built from another unit
// (input_ref) whose content no longer hashes to what was recorded (input_sha).
// An input_ref naming the story is hashed from the project's story, one naming
// a beat from that beat in project_beat, and one naming a beat's dialogue from
// that row of project_script.
var reviewUnitColumns = `ru.project_id, ru.kind, ru.unit_key, ru.status, ru.content, ru.creator_fields, ru.redo_count, ru.updated_at,
    ru.input_ref, ru.input_sha,
    (ru.input_ref <> '' AND ru.input_sha <> COALESCE((
        SELECT md5(src.content::text) FROM review_units src
        WHERE src.project_id = ru.project_id AND src.kind || ':' || src.unit_key = ru.input_ref
    ), (
        SELECT md5(ps.central_question || E'\n' || ps.core_answer || E'\n' || ps.narrative_arc::text)
        FROM project_story ps WHERE ps.project_id = ru.project_id AND ru.input_ref = '` + domain.StoryInputKind + `:` + domain.StoryInputKey + `'
    ), (
        SELECT md5(` + beatContentJSON("pb") + `) FROM project_beat pb
        WHERE pb.project_id = ru.project_id AND ru.input_ref = '` + domain.UnitKindBeat + `:' || pb.beat_key
    ), (
        SELECT md5(` + scriptContentJSON("sc") + `::text) FROM project_script sc
        WHERE sc.project_id = ru.project_id AND ru.input_ref = '` + domain.UnitKindScriptBeat + `:' || sc.beat_key
    ), ''))`

func scanReviewUnit(row pgx.Row) (domain.ReviewUnit, error) {
	var u domain.ReviewUnit
	var status string
	err := row.Scan(&u.ProjectID, &u.Kind, &u.Key, &status, &u.Content, &u.CreatorFields, &u.RedoCount, &u.UpdatedAt,
		&u.InputRef, &u.InputSHA, &u.Stale)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	u.Status = domain.UnitStatus(status)
	if u.CreatorFields == nil {
		u.CreatorFields = []string{}
	}
	return u, nil
}

// GetUnit returns one review unit, or domain.ErrUnitNotFound.
func (r *PromptTemplateRepository) GetUnit(ctx context.Context, projectID, kind, key string) (domain.ReviewUnit, error) {
	if kind == domain.UnitKindScriptBeat {
		return r.getScriptUnit(ctx, projectID, key)
	}
	if isLockKind(kind) {
		return r.getLockUnit(ctx, projectID, kind, key)
	}
	if kind == domain.UnitKindResearch {
		return r.getResearchGateUnit(ctx, projectID, key)
	}
	if kind == domain.UnitKindResearchQuestion {
		return r.getResearchQuestionUnit(ctx, projectID, key)
	}
	return scanReviewUnit(r.pool.QueryRow(ctx,
		`SELECT `+reviewUnitColumns+` FROM review_units ru WHERE ru.project_id = $1 AND ru.kind = $2 AND ru.unit_key = $3`,
		projectID, kind, key))
}

// ListUnits returns every unit of a kind, ordered so that Q2 comes before Q10.
func (r *PromptTemplateRepository) ListUnits(ctx context.Context, projectID, kind string) ([]domain.ReviewUnit, error) {
	if kind == domain.UnitKindScriptBeat {
		return r.listScriptUnits(ctx, projectID)
	}
	if isLockKind(kind) {
		return r.listLockUnits(ctx, projectID, kind)
	}
	if kind == domain.UnitKindResearch {
		u, err := r.getResearchGateUnit(ctx, projectID, domain.ResearchUnitKey)
		if errors.Is(err, domain.ErrUnitNotFound) {
			return []domain.ReviewUnit{}, nil
		}
		if err != nil {
			return nil, err
		}
		return []domain.ReviewUnit{u}, nil
	}
	if kind == domain.UnitKindResearchQuestion {
		return r.listResearchQuestionUnits(ctx, projectID)
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+reviewUnitColumns+` FROM review_units ru WHERE ru.project_id = $1 AND ru.kind = $2
		 ORDER BY length(ru.unit_key), ru.unit_key`,
		projectID, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	units := []domain.ReviewUnit{}
	for rows.Next() {
		u, err := scanReviewUnit(rows)
		if err != nil {
			return nil, err
		}
		units = append(units, u)
	}
	return units, rows.Err()
}

// DeleteUnit removes one unit; a unit that is not there is not an error.
func (r *PromptTemplateRepository) DeleteUnit(ctx context.Context, projectID, kind, key string) error {
	if kind == domain.UnitKindScriptBeat {
		return r.deleteScriptUnit(ctx, projectID, key)
	}
	if isLockKind(kind) {
		return r.deleteLockUnit(ctx, projectID, kind, key)
	}
	if kind == domain.UnitKindResearch {
		_, err := r.pool.Exec(ctx, `DELETE FROM project_research_gate WHERE project_id = $1`, projectID)
		return err
	}
	if kind == domain.UnitKindResearchQuestion {
		return r.deleteResearchQuestionUnit(ctx, projectID, key)
	}
	_, err := r.pool.Exec(ctx,
		`DELETE FROM review_units WHERE project_id = $1 AND kind = $2 AND unit_key = $3`, projectID, kind, key)
	return err
}

// UnitSHA returns the md5 of a unit's content, or "" when there is no such
// unit. It is the same hash the stale column compares against.
func (r *PromptTemplateRepository) UnitSHA(ctx context.Context, projectID, kind, key string) (string, error) {
	if isLockKind(kind) {
		return r.lockUnitSHA(ctx, projectID, kind, key)
	}
	if kind == domain.UnitKindResearchQuestion {
		return r.researchQuestionSHA(ctx, projectID, key)
	}
	query := `SELECT md5(content::text) FROM review_units WHERE project_id = $1 AND kind = $2 AND unit_key = $3`
	args := []any{projectID, kind, key}
	switch {
	case kind == domain.StoryInputKind && key == domain.StoryInputKey:
		query, args = `SELECT `+storyContentSHA, []any{projectID}
	case kind == domain.UnitKindBeat:
		query, args = `SELECT `+beatContentSHA, []any{projectID, key}
	case kind == domain.UnitKindScriptBeat:
		query, args = `SELECT `+scriptContentSHA, []any{projectID, key}
	}
	var sha *string
	err := r.pool.QueryRow(ctx, query, args...).Scan(&sha)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && sha == nil) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return *sha, nil
}

// SaveUnit upserts a review unit and returns it as stored.
func (r *PromptTemplateRepository) SaveUnit(ctx context.Context, unit domain.ReviewUnit) (domain.ReviewUnit, error) {
	if unit.Kind == domain.UnitKindScriptBeat {
		return r.saveScriptUnit(ctx, unit)
	}
	if isLockKind(unit.Kind) {
		return r.saveLockUnit(ctx, unit)
	}
	if unit.Kind == domain.UnitKindResearch {
		return r.saveResearchGateUnit(ctx, unit)
	}
	if unit.Kind == domain.UnitKindResearchQuestion {
		return r.saveResearchQuestionUnit(ctx, unit)
	}
	creatorFields := unit.CreatorFields
	if creatorFields == nil {
		creatorFields = []string{}
	}
	content := unit.Content
	if len(content) == 0 {
		content = []byte("{}")
	}
	return scanReviewUnit(r.pool.QueryRow(ctx, `
		INSERT INTO review_units AS ru (project_id, kind, unit_key, status, content, creator_fields, redo_count, input_ref, input_sha, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
		ON CONFLICT (project_id, kind, unit_key) DO UPDATE SET
		    status = EXCLUDED.status,
		    content = EXCLUDED.content,
		    creator_fields = EXCLUDED.creator_fields,
		    redo_count = EXCLUDED.redo_count,
		    input_ref = EXCLUDED.input_ref,
		    input_sha = EXCLUDED.input_sha,
		    updated_at = now()
		RETURNING `+reviewUnitColumns,
		unit.ProjectID, unit.Kind, unit.Key, string(unit.Status), content, creatorFields, unit.RedoCount,
		unit.InputRef, unit.InputSHA))
}
