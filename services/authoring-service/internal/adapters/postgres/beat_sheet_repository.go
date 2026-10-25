package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"authoring/internal/domain"
)

// beatContentJSON is the SQL text of the content of the beat row aliased a: the
// eight fields a later step hashes, in a form that does not depend on column
// order or on the review state beside them.
func beatContentJSON(a string) string {
	return `jsonb_build_object('fn', ` + a + `.fn, 'title', ` + a + `.title, 'goal', ` + a + `.goal, 'content', ` + a + `.content,
        'facts', to_jsonb(` + a + `.facts), 'dur', ` + a + `.dur, 'next', ` + a + `.next, 'step', ` + a + `.step)::text`
}

// beatContentSHA hashes the content of beat $2 of project $1: the hash a later
// step records as input_sha when it names the beat as its input (see
// domain.UnitKindBeat), so that dialogue written from a beat that has changed
// since reads as stale. Only the beat's content is hashed, not its review state.
var beatContentSHA = `(SELECT md5(` + beatContentJSON("pb") + `)
    FROM project_beat pb WHERE pb.project_id = $1 AND pb.beat_key = $2)`

const beatColumns = `beat_key, status, creator_fields, redo_count, story_hash,
    fn, title, goal, content, facts, dur, next, step`

// GetBeatSheet returns a project's beat sheet, or domain.ErrBeatSheetNotFound.
func (r *PromptTemplateRepository) GetBeatSheet(ctx context.Context, projectID string) (domain.StoredBeatSheet, error) {
	sheet := domain.StoredBeatSheet{ProjectID: projectID}
	err := r.pool.QueryRow(ctx, `
		SELECT next_seq, skeleton_approved, rebalanced, updated_at
		FROM project_beat_sheet WHERE project_id = $1`, projectID).Scan(
		&sheet.NextSeq, &sheet.SkeletonApproved, &sheet.Rebalanced, &sheet.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.StoredBeatSheet{}, domain.ErrBeatSheetNotFound
	}
	if err != nil {
		return domain.StoredBeatSheet{}, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+beatColumns+` FROM project_beat WHERE project_id = $1 ORDER BY position`, projectID)
	if err != nil {
		return domain.StoredBeatSheet{}, err
	}
	defer rows.Close()
	sheet.Beats = []domain.StoredBeat{}
	for rows.Next() {
		var b domain.StoredBeat
		var status string
		if err := rows.Scan(&b.Key, &status, &b.CreatorFields, &b.RedoCount, &b.StoryHash,
			&b.Beat.Fn, &b.Beat.Title, &b.Beat.Goal, &b.Beat.Content, &b.Beat.Facts, &b.Beat.Dur, &b.Beat.Next, &b.Beat.Step); err != nil {
			return domain.StoredBeatSheet{}, err
		}
		b.Status = domain.UnitStatus(status)
		if b.CreatorFields == nil {
			b.CreatorFields = []string{}
		}
		if b.Beat.Facts == nil {
			b.Beat.Facts = []string{}
		}
		sheet.Beats = append(sheet.Beats, b)
	}
	return sheet, rows.Err()
}

// SaveBeatSheet inserts or replaces a project's beat sheet: the sheet row and
// every beat, in one transaction, the position of a beat being its place in
// sheet.Beats.
func (r *PromptTemplateRepository) SaveBeatSheet(ctx context.Context, sheet domain.StoredBeatSheet) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		INSERT INTO project_beat_sheet (project_id, next_seq, skeleton_approved, rebalanced, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (project_id) DO UPDATE SET
		    next_seq = EXCLUDED.next_seq, skeleton_approved = EXCLUDED.skeleton_approved,
		    rebalanced = EXCLUDED.rebalanced, updated_at = EXCLUDED.updated_at`,
		sheet.ProjectID, sheet.NextSeq, sheet.SkeletonApproved, sheet.Rebalanced); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM project_beat WHERE project_id = $1`, sheet.ProjectID); err != nil {
		return err
	}
	for i, b := range sheet.Beats {
		fields, facts := b.CreatorFields, b.Beat.Facts
		if fields == nil {
			fields = []string{}
		}
		if facts == nil {
			facts = []string{}
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO project_beat (project_id, beat_key, position, fn, title, goal, content, facts, dur, next, step,
			                          status, creator_fields, redo_count, story_hash, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, now())`,
			sheet.ProjectID, b.Key, i+1, b.Beat.Fn, b.Beat.Title, b.Beat.Goal, b.Beat.Content, facts, b.Beat.Dur,
			b.Beat.Next, b.Beat.Step, string(b.Status), fields, b.RedoCount, b.StoryHash); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
