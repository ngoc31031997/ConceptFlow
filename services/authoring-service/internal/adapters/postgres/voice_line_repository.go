package postgres

import (
	"context"
	"time"

	"authoring/internal/domain"
)

const voiceLineColumns = `project_id, line_id, text, text_sha, voice_id, language, status, audio_path, duration_s, cache_key, cache_hit, error, requested_at`

// ListVoiceLines returns every voice line of a project.
func (r *PromptTemplateRepository) ListVoiceLines(ctx context.Context, projectID string) ([]domain.VoiceLine, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+voiceLineColumns+` FROM voice_lines WHERE project_id = $1 ORDER BY line_id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.VoiceLine{}
	for rows.Next() {
		var l domain.VoiceLine
		var status string
		if err := rows.Scan(&l.ProjectID, &l.LineID, &l.Text, &l.TextSHA, &l.VoiceID, &l.Language, &status, &l.AudioPath,
			&l.DurationSeconds, &l.CacheKey, &l.CacheHit, &l.Error, &l.RequestedAt); err != nil {
			return nil, err
		}
		l.Status = domain.VoiceLineStatus(status)
		out = append(out, l)
	}
	return out, rows.Err()
}

// RequestVoiceLine sets a line to pending with the given wording and voice. The earlier
// file of the line is forgotten: it was read from other wording or another voice.
func (r *PromptTemplateRepository) RequestVoiceLine(ctx context.Context, l domain.VoiceLine) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO voice_lines (project_id, line_id, text, text_sha, voice_id, language, status, requested_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending', now())
		ON CONFLICT (project_id, line_id) DO UPDATE SET
		    text = EXCLUDED.text, text_sha = EXCLUDED.text_sha, voice_id = EXCLUDED.voice_id, language = EXCLUDED.language,
		    status = 'pending', audio_path = '', duration_s = 0, cache_key = '', cache_hit = false, error = '', requested_at = now()
	`, l.ProjectID, l.LineID, l.Text, l.TextSHA, l.VoiceID, l.Language)
	return err
}

// ClaimVoiceLines returns the lines waiting to be read, and those running for longer than
// retryAfter, and marks them running. One statement, so two callers never get the same line.
func (r *PromptTemplateRepository) ClaimVoiceLines(ctx context.Context, projectID string, retryAfter time.Duration) ([]domain.VoiceLine, error) {
	rows, err := r.pool.Query(ctx, `
		UPDATE voice_lines SET status = 'running', requested_at = now()
		WHERE project_id = $1 AND (status = 'pending' OR (status = 'running' AND requested_at < now() - make_interval(secs => $2)))
		RETURNING `+voiceLineColumns, projectID, retryAfter.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.VoiceLine{}
	for rows.Next() {
		var l domain.VoiceLine
		var status string
		if err := rows.Scan(&l.ProjectID, &l.LineID, &l.Text, &l.TextSHA, &l.VoiceID, &l.Language, &status, &l.AudioPath,
			&l.DurationSeconds, &l.CacheKey, &l.CacheHit, &l.Error, &l.RequestedAt); err != nil {
			return nil, err
		}
		l.Status = domain.VoiceLineStatus(status)
		out = append(out, l)
	}
	return out, rows.Err()
}

// SaveVoiceResults records what the voice service reported. A result is applied only
// while the line still has the wording it was sent with (ref = text_sha).
func (r *PromptTemplateRepository) SaveVoiceResults(ctx context.Context, projectID string, results []domain.VoiceLineResult) error {
	for _, res := range results {
		status := "ready"
		if res.Error != "" {
			status = "failed"
		}
		if _, err := r.pool.Exec(ctx, `
			UPDATE voice_lines SET status = $4, audio_path = $5, duration_s = $6, cache_key = $7, cache_hit = $8, error = $9
			WHERE project_id = $1 AND line_id = $2 AND text_sha = $3
		`, projectID, res.LineID, res.Ref, status, res.AudioPath, res.DurationSeconds, res.CacheKey, res.CacheHit, res.Error); err != nil {
			return err
		}
	}
	return nil
}

// FailVoiceLines marks the lines failed with the reason.
func (r *PromptTemplateRepository) FailVoiceLines(ctx context.Context, projectID string, lineIDs []string, reason string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE voice_lines SET status = 'failed', error = $3 WHERE project_id = $1 AND line_id = ANY($2)
	`, projectID, lineIDs, reason)
	return err
}
