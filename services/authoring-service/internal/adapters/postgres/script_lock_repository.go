package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"authoring/internal/domain"
)

// The lock step keeps its own rows: the review of the script in
// project_script_review, the handoff note of each beat in project_handoff and
// the glossary in project_glossary, one row per term. The step reads and
// writes them as review units, so the unit store routes those kinds here.

// isLockKind reports whether a unit kind is stored by the lock step's tables.
func isLockKind(kind string) bool {
	return kind == domain.UnitKindScriptReview || kind == domain.UnitKindHandoffBeat || kind == domain.UnitKindGlossary
}

// handoffContentJSON is the SQL text of the content of the handoff row aliased
// a, as a review unit presents it (the five sections of the note).
func handoffContentJSON(a string) string {
	return `jsonb_build_object('must', ` + a + `.must, 'terms', ` + a + `.terms, 'exact', ` + a + `.exact,
        'free', ` + a + `.free, 'unsure', ` + a + `.unsure)`
}

// handoffUnitColumns lists a handoff row as the columns of a review unit (see
// reviewUnitColumns), the dialogue it was written from deciding the stale flag.
var handoffUnitColumns = `ph.project_id, '` + domain.UnitKindHandoffBeat + `', ph.beat_key, ph.status, ` + handoffContentJSON("ph") + `,
    ph.creator_fields, ph.redo_count, ph.updated_at, ph.input_ref, ph.input_sha,
    (ph.input_ref <> '' AND ph.input_sha <> COALESCE((
        SELECT md5(` + scriptContentJSON("sc") + `::text) FROM project_script sc
        WHERE sc.project_id = ph.project_id AND ph.input_ref = '` + domain.UnitKindScriptBeat + `:' || sc.beat_key
    ), ''))`

func (r *PromptTemplateRepository) getLockUnit(ctx context.Context, projectID, kind, key string) (domain.ReviewUnit, error) {
	switch kind {
	case domain.UnitKindHandoffBeat:
		return scanReviewUnit(r.pool.QueryRow(ctx,
			`SELECT `+handoffUnitColumns+` FROM project_handoff ph WHERE ph.project_id = $1 AND ph.beat_key = $2`, projectID, key))
	case domain.UnitKindScriptReview:
		if key != domain.ScriptReviewUnitKey {
			return domain.ReviewUnit{}, domain.ErrUnitNotFound
		}
		return r.getScriptReviewUnit(ctx, projectID)
	default:
		if key != domain.GlossaryUnitKey {
			return domain.ReviewUnit{}, domain.ErrUnitNotFound
		}
		return r.getGlossaryUnit(ctx, projectID)
	}
}

func (r *PromptTemplateRepository) listLockUnits(ctx context.Context, projectID, kind string) ([]domain.ReviewUnit, error) {
	if kind != domain.UnitKindHandoffBeat {
		key := domain.ScriptReviewUnitKey
		if kind == domain.UnitKindGlossary {
			key = domain.GlossaryUnitKey
		}
		u, err := r.getLockUnit(ctx, projectID, kind, key)
		if errors.Is(err, domain.ErrUnitNotFound) {
			return []domain.ReviewUnit{}, nil
		}
		if err != nil {
			return nil, err
		}
		return []domain.ReviewUnit{u}, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+handoffUnitColumns+` FROM project_handoff ph WHERE ph.project_id = $1 ORDER BY length(ph.beat_key), ph.beat_key`, projectID)
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

func (r *PromptTemplateRepository) deleteLockUnit(ctx context.Context, projectID, kind, key string) error {
	var err error
	switch kind {
	case domain.UnitKindHandoffBeat:
		_, err = r.pool.Exec(ctx, `DELETE FROM project_handoff WHERE project_id = $1 AND beat_key = $2`, projectID, key)
	case domain.UnitKindScriptReview:
		_, err = r.pool.Exec(ctx, `DELETE FROM project_script_review WHERE project_id = $1`, projectID)
	default:
		_, err = r.pool.Exec(ctx, `DELETE FROM project_glossary WHERE project_id = $1`, projectID)
	}
	return err
}

// lockUnitSHA hashes a lock unit's content as the unit presents it, or "" when there is none.
func (r *PromptTemplateRepository) lockUnitSHA(ctx context.Context, projectID, kind, key string) (string, error) {
	u, err := r.getLockUnit(ctx, projectID, kind, key)
	if errors.Is(err, domain.ErrUnitNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var sha string
	err = r.pool.QueryRow(ctx, `SELECT md5($1::jsonb::text)`, string(u.Content)).Scan(&sha)
	return sha, err
}

func (r *PromptTemplateRepository) saveLockUnit(ctx context.Context, unit domain.ReviewUnit) (domain.ReviewUnit, error) {
	fields := unit.CreatorFields
	if fields == nil {
		fields = []string{}
	}
	switch unit.Kind {
	case domain.UnitKindHandoffBeat:
		var n domain.HandoffNote
		if len(unit.Content) > 0 {
			if err := json.Unmarshal(unit.Content, &n); err != nil {
				return domain.ReviewUnit{}, fmt.Errorf("read handoff of %s: %w", unit.Key, err)
			}
		}
		n = n.Normalized()
		sections := make([][]byte, 0, 5)
		for _, s := range [][]string{n.Must, n.Terms, n.Exact, n.Free, n.Unsure} {
			raw, err := json.Marshal(s)
			if err != nil {
				return domain.ReviewUnit{}, err
			}
			sections = append(sections, raw)
		}
		if _, err := r.pool.Exec(ctx, `
			INSERT INTO project_handoff (project_id, beat_key, must, terms, exact, free, unsure,
			                             status, creator_fields, redo_count, input_ref, input_sha, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, now())
			ON CONFLICT (project_id, beat_key) DO UPDATE SET
			    must = EXCLUDED.must, terms = EXCLUDED.terms, exact = EXCLUDED.exact, free = EXCLUDED.free,
			    unsure = EXCLUDED.unsure, status = EXCLUDED.status, creator_fields = EXCLUDED.creator_fields,
			    redo_count = EXCLUDED.redo_count, input_ref = EXCLUDED.input_ref, input_sha = EXCLUDED.input_sha,
			    updated_at = now()`,
			unit.ProjectID, unit.Key, sections[0], sections[1], sections[2], sections[3], sections[4],
			string(unit.Status), fields, unit.RedoCount, unit.InputRef, unit.InputSHA); err != nil {
			return domain.ReviewUnit{}, err
		}
		return r.getLockUnit(ctx, unit.ProjectID, unit.Kind, unit.Key)
	case domain.UnitKindScriptReview:
		if err := r.saveScriptReview(ctx, unit, fields); err != nil {
			return domain.ReviewUnit{}, err
		}
		return r.getScriptReviewUnit(ctx, unit.ProjectID)
	default:
		if err := r.saveGlossary(ctx, unit); err != nil {
			return domain.ReviewUnit{}, err
		}
		u, err := r.getGlossaryUnit(ctx, unit.ProjectID)
		if errors.Is(err, domain.ErrUnitNotFound) {
			u = domain.ReviewUnit{ProjectID: unit.ProjectID, Kind: unit.Kind, Key: domain.GlossaryUnitKey,
				Status: domain.UnitPending, Content: []byte(`{"terms":[]}`), CreatorFields: []string{}}
			err = nil
		}
		return u, err
	}
}

// --- review -------------------------------------------------------------------

func (r *PromptTemplateRepository) getScriptReviewUnit(ctx context.Context, projectID string) (domain.ReviewUnit, error) {
	u := domain.ReviewUnit{ProjectID: projectID, Kind: domain.UnitKindScriptReview, Key: domain.ScriptReviewUnitKey}
	var rev domain.ScriptReview
	var findings, decisions, aiNotes, myNotes []byte
	var status string
	err := r.pool.QueryRow(ctx, `
		SELECT findings, next_seq, decisions, ai_notes, my_notes, reviewed_sha, rounds,
		       status, creator_fields, redo_count, updated_at
		FROM project_script_review WHERE project_id = $1`, projectID).Scan(
		&findings, &rev.NextSeq, &decisions, &aiNotes, &myNotes, &rev.ReviewedSHA, &rev.Rounds,
		&status, &u.CreatorFields, &u.RedoCount, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	for _, p := range []struct {
		raw []byte
		to  any
	}{{findings, &rev.Findings}, {decisions, &rev.Decisions}, {aiNotes, &rev.AINotes}, {myNotes, &rev.MyNotes}} {
		if err := json.Unmarshal(p.raw, p.to); err != nil {
			return domain.ReviewUnit{}, fmt.Errorf("read stored script review: %w", err)
		}
	}
	if u.Content, err = json.Marshal(rev.Normalized()); err != nil {
		return domain.ReviewUnit{}, err
	}
	u.Status = domain.UnitStatus(status)
	if u.CreatorFields == nil {
		u.CreatorFields = []string{}
	}
	return u, nil
}

func (r *PromptTemplateRepository) saveScriptReview(ctx context.Context, unit domain.ReviewUnit, fields []string) error {
	var rev domain.ScriptReview
	if len(unit.Content) > 0 {
		if err := json.Unmarshal(unit.Content, &rev); err != nil {
			return fmt.Errorf("read script review: %w", err)
		}
	}
	rev = rev.Normalized()
	cols := make([][]byte, 0, 4)
	for _, v := range []any{rev.Findings, rev.Decisions, rev.AINotes, rev.MyNotes} {
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		cols = append(cols, raw)
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO project_script_review (project_id, findings, next_seq, decisions, ai_notes, my_notes, reviewed_sha, rounds,
		                                   status, creator_fields, redo_count, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, now())
		ON CONFLICT (project_id) DO UPDATE SET
		    findings = EXCLUDED.findings, next_seq = EXCLUDED.next_seq, decisions = EXCLUDED.decisions,
		    ai_notes = EXCLUDED.ai_notes, my_notes = EXCLUDED.my_notes, reviewed_sha = EXCLUDED.reviewed_sha,
		    rounds = EXCLUDED.rounds, status = EXCLUDED.status, creator_fields = EXCLUDED.creator_fields,
		    redo_count = EXCLUDED.redo_count, updated_at = now()`,
		unit.ProjectID, cols[0], rev.NextSeq, cols[1], cols[2], cols[3], rev.ReviewedSHA, rev.Rounds,
		string(unit.Status), fields, unit.RedoCount)
	return err
}

// --- glossary -----------------------------------------------------------------

func (r *PromptTemplateRepository) getGlossaryUnit(ctx context.Context, projectID string) (domain.ReviewUnit, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT term, say, draw, avoid, mine, updated_at FROM project_glossary
		WHERE project_id = $1 ORDER BY position`, projectID)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	defer rows.Close()
	g := domain.Glossary{Terms: []domain.GlossaryTerm{}}
	var updated time.Time
	for rows.Next() {
		var t domain.GlossaryTerm
		var at time.Time
		if err := rows.Scan(&t.Term, &t.Say, &t.Draw, &t.Avoid, &t.Mine, &at); err != nil {
			return domain.ReviewUnit{}, err
		}
		if at.After(updated) {
			updated = at
		}
		g.Terms = append(g.Terms, t)
	}
	if err := rows.Err(); err != nil {
		return domain.ReviewUnit{}, err
	}
	if len(g.Terms) == 0 {
		return domain.ReviewUnit{}, domain.ErrUnitNotFound
	}
	raw, err := json.Marshal(g)
	if err != nil {
		return domain.ReviewUnit{}, err
	}
	return domain.ReviewUnit{
		ProjectID: projectID, Kind: domain.UnitKindGlossary, Key: domain.GlossaryUnitKey,
		Status: domain.UnitPending, Content: raw, CreatorFields: []string{}, UpdatedAt: updated,
	}, nil
}

// saveGlossary replaces the project's terms with the unit's, in one transaction.
func (r *PromptTemplateRepository) saveGlossary(ctx context.Context, unit domain.ReviewUnit) error {
	var g domain.Glossary
	if len(unit.Content) > 0 {
		if err := json.Unmarshal(unit.Content, &g); err != nil {
			return fmt.Errorf("read glossary: %w", err)
		}
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM project_glossary WHERE project_id = $1`, unit.ProjectID); err != nil {
		return err
	}
	for i, t := range g.Terms {
		if _, err := tx.Exec(ctx, `
			INSERT INTO project_glossary (project_id, position, term, say, draw, avoid, mine, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, now())`,
			unit.ProjectID, i+1, t.Term, t.Say, t.Draw, t.Avoid, t.Mine); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
