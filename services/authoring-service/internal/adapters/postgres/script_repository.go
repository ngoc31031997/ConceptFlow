package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"authoring/internal/domain"
)

// scriptContentJSON is the SQL text of the content of the dialogue row aliased
// a, as a review unit presents it ({lines, next_seq}).
func scriptContentJSON(a string) string {
	return `jsonb_build_object('lines', ` + a + `.lines, 'next_seq', ` + a + `.next_seq)`
}

// scriptUnitColumns lists a dialogue row as the columns of a review unit (see
// reviewUnitColumns), the beat it was written from deciding the stale flag.
var scriptUnitColumns = `sc.project_id, '` + domain.UnitKindScriptBeat + `', sc.beat_key, sc.status, ` + scriptContentJSON("sc") + `,
    sc.creator_fields, sc.redo_count, sc.updated_at, sc.input_ref, sc.input_sha,
    (sc.input_ref <> '' AND sc.input_sha <> COALESCE((
        SELECT md5(` + beatContentJSON("pb") + `) FROM project_beat pb
        WHERE pb.project_id = sc.project_id AND sc.input_ref = '` + domain.UnitKindBeat + `:' || pb.beat_key
    ), ''))`

// scriptContentSHA hashes the dialogue of beat $2 of project $1, the form a
// step built from it (its handoff note) records as input_sha.
const scriptContentSHA = `(SELECT md5(` + `jsonb_build_object('lines', sc.lines, 'next_seq', sc.next_seq)::text` + `)
    FROM project_script sc WHERE sc.project_id = $1 AND sc.beat_key = $2)`

func (r *PromptTemplateRepository) getScriptUnit(ctx context.Context, projectID, key string) (domain.ReviewUnit, error) {
	return scanReviewUnit(r.pool.QueryRow(ctx,
		`SELECT `+scriptUnitColumns+` FROM project_script sc WHERE sc.project_id = $1 AND sc.beat_key = $2`, projectID, key))
}

func (r *PromptTemplateRepository) listScriptUnits(ctx context.Context, projectID string) ([]domain.ReviewUnit, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+scriptUnitColumns+` FROM project_script sc WHERE sc.project_id = $1 ORDER BY length(sc.beat_key), sc.beat_key`, projectID)
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

func (r *PromptTemplateRepository) deleteScriptUnit(ctx context.Context, projectID, key string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM project_script WHERE project_id = $1 AND beat_key = $2`, projectID, key)
	return err
}

func (r *PromptTemplateRepository) saveScriptUnit(ctx context.Context, unit domain.ReviewUnit) (domain.ReviewUnit, error) {
	script := domain.ScriptBeat{NextSeq: 1}
	if len(unit.Content) > 0 {
		if err := json.Unmarshal(unit.Content, &script); err != nil {
			return domain.ReviewUnit{}, fmt.Errorf("read dialogue of %s: %w", unit.Key, err)
		}
	}
	if script.Lines == nil {
		script.Lines = []domain.ScriptLine{}
	}
	lines, err := json.Marshal(script.Lines)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	fields := unit.CreatorFields
	if fields == nil {
		fields = []string{}
	}
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO project_script (project_id, beat_key, lines, next_seq, status, creator_fields, redo_count, input_ref, input_sha, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
		ON CONFLICT (project_id, beat_key) DO UPDATE SET
		    lines = EXCLUDED.lines, next_seq = EXCLUDED.next_seq, status = EXCLUDED.status,
		    creator_fields = EXCLUDED.creator_fields, redo_count = EXCLUDED.redo_count,
		    input_ref = EXCLUDED.input_ref, input_sha = EXCLUDED.input_sha, updated_at = now()`,
		unit.ProjectID, unit.Key, lines, script.NextSeq, string(unit.Status), fields, unit.RedoCount,
		unit.InputRef, unit.InputSHA); err != nil {
		return domain.ReviewUnit{}, err
	}
	return r.getScriptUnit(ctx, unit.ProjectID, unit.Key)
}
