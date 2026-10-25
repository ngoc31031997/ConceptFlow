package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"authoring/internal/domain"
)

// beatContentSHA hashes the content of beat $2 of project $1: the hash a later
// step records as input_sha when it names the beat as its input (see
// domain.UnitKindBeat), so that dialogue written from a beat that has changed
// since reads as stale. Only the beat's content is hashed, not its review state.
const beatContentSHA = `(SELECT md5((elem->'beat')::text)
    FROM project_beat_sheet pbs, jsonb_array_elements(pbs.beats) elem
    WHERE pbs.project_id = $1 AND elem->>'key' = $2)`

// GetBeatSheet returns a project's beat sheet, or domain.ErrBeatSheetNotFound.
func (r *PromptTemplateRepository) GetBeatSheet(ctx context.Context, projectID string) (domain.StoredBeatSheet, error) {
	sheet := domain.StoredBeatSheet{ProjectID: projectID}
	var beats []byte
	err := r.pool.QueryRow(ctx, `
		SELECT beats, next_seq, skeleton_approved, rebalanced, updated_at
		FROM project_beat_sheet WHERE project_id = $1`, projectID).Scan(
		&beats, &sheet.NextSeq, &sheet.SkeletonApproved, &sheet.Rebalanced, &sheet.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.StoredBeatSheet{}, domain.ErrBeatSheetNotFound
	}
	if err != nil {
		return domain.StoredBeatSheet{}, err
	}
	if err := json.Unmarshal(beats, &sheet.Beats); err != nil {
		return domain.StoredBeatSheet{}, fmt.Errorf("read beats: %w", err)
	}
	for i := range sheet.Beats {
		if sheet.Beats[i].CreatorFields == nil {
			sheet.Beats[i].CreatorFields = []string{}
		}
		if sheet.Beats[i].Beat.Facts == nil {
			sheet.Beats[i].Beat.Facts = []string{}
		}
	}
	return sheet, nil
}

// SaveBeatSheet inserts or replaces a project's beat sheet, all of it in one statement.
func (r *PromptTemplateRepository) SaveBeatSheet(ctx context.Context, sheet domain.StoredBeatSheet) error {
	beats := sheet.Beats
	if beats == nil {
		beats = []domain.StoredBeat{}
	}
	for i := range beats {
		if beats[i].CreatorFields == nil {
			beats[i].CreatorFields = []string{}
		}
		if beats[i].Beat.Facts == nil {
			beats[i].Beat.Facts = []string{}
		}
	}
	raw, err := json.Marshal(beats)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO project_beat_sheet (project_id, beats, next_seq, skeleton_approved, rebalanced, updated_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (project_id) DO UPDATE SET
		    beats = EXCLUDED.beats, next_seq = EXCLUDED.next_seq,
		    skeleton_approved = EXCLUDED.skeleton_approved, rebalanced = EXCLUDED.rebalanced,
		    updated_at = EXCLUDED.updated_at`,
		sheet.ProjectID, raw, sheet.NextSeq, sheet.SkeletonApproved, sheet.Rebalanced)
	return err
}
