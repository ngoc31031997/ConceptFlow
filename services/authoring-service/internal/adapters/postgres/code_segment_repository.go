package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"authoring/internal/domain"
)

// The code step's segments (ADR-0030), the Creator's shots
// per segment, and the failed-check log.

const segmentColumns = `key, kind, position, shots, status, source, fingerprint, content,
	error_kind, error_message, duration_ms, updated_at`

func scanSegment(row pgx.Row) (domain.CodeSegment, error) {
	var s domain.CodeSegment
	var status string
	var content []byte
	if err := row.Scan(&s.Key, &s.Kind, &s.Position, &s.Shots, &status, &s.Source, &s.Fingerprint, &content,
		&s.ErrorKind, &s.ErrorMessage, &s.DurationMS, &s.UpdatedAt); err != nil {
		return s, err
	}
	s.Status = domain.SegmentStatus(status)
	if len(content) > 0 {
		s.Content = content
	}
	if s.Shots == nil {
		s.Shots = []string{}
	}
	return s, nil
}

func (r *PromptTemplateRepository) ListSegments(ctx context.Context, projectID, step string) ([]domain.CodeSegment, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+segmentColumns+`
		FROM authoring_segments WHERE project_id = $1 AND step = $2 ORDER BY position, key`, projectID, step)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.CodeSegment{}
	for rows.Next() {
		s, err := scanSegment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ApplySegmentPlan makes the stored segments follow the plan llm-service sent
// at the start of a run: keys the plan no longer has are deleted, new keys are
// added as pending, and a segment whose fingerprint changed goes back to
// pending without its content. A matching segment is untouched.
func (r *PromptTemplateRepository) ApplySegmentPlan(ctx context.Context, projectID, step string, plan []domain.CodeSegment) error {
	keys := make([]string, 0, len(plan))
	for _, s := range plan {
		keys = append(keys, s.Key)
	}
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM authoring_segments
			WHERE project_id = $1 AND step = $2 AND NOT (key = ANY($3))`, projectID, step, keys); err != nil {
			return err
		}
		for _, s := range plan {
			shots := s.Shots
			if shots == nil {
				shots = []string{}
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO authoring_segments (project_id, step, key, position, kind, shots, fingerprint)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				ON CONFLICT (project_id, step, key) DO UPDATE SET
					position = EXCLUDED.position, kind = EXCLUDED.kind, shots = EXCLUDED.shots,
					status = CASE WHEN authoring_segments.fingerprint = EXCLUDED.fingerprint
						THEN authoring_segments.status ELSE 'pending' END,
					content = CASE WHEN authoring_segments.fingerprint = EXCLUDED.fingerprint
						THEN authoring_segments.content ELSE NULL END,
					source = CASE WHEN authoring_segments.fingerprint = EXCLUDED.fingerprint
						THEN authoring_segments.source ELSE '' END,
					error_kind = CASE WHEN authoring_segments.fingerprint = EXCLUDED.fingerprint
						THEN authoring_segments.error_kind ELSE '' END,
					error_message = CASE WHEN authoring_segments.fingerprint = EXCLUDED.fingerprint
						THEN authoring_segments.error_message ELSE '' END,
					updated_at = CASE WHEN authoring_segments.fingerprint = EXCLUDED.fingerprint
						THEN authoring_segments.updated_at ELSE now() END,
					fingerprint = EXCLUDED.fingerprint
			`, projectID, step, s.Key, s.Position, s.Kind, shots, s.Fingerprint); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PromptTemplateRepository) MarkSegmentRunning(ctx context.Context, projectID, step, key string) error {
	return r.updateSegment(ctx, `UPDATE authoring_segments
		SET status = 'running', error_kind = '', error_message = '', updated_at = now()
		WHERE project_id = $1 AND step = $2 AND key = $3`, projectID, step, key)
}

// SaveSegmentDone stores a segment's final content (Q8: one result, the new
// one overwrites). A repaired segment keeps its source and duration: the
// repair fixed what the AI, the outside AI or the Creator wrote.
func (r *PromptTemplateRepository) SaveSegmentDone(ctx context.Context, projectID, step string, s domain.CodeSegment, repaired bool) error {
	shots := s.Shots
	if shots == nil {
		shots = []string{}
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO authoring_segments (project_id, step, key, position, kind, shots, status, source, fingerprint, content, duration_ms)
		VALUES ($1, $2, $3, $4, $5, $6, 'done', $7, $8, $9, $10)
		ON CONFLICT (project_id, step, key) DO UPDATE SET
			status = 'done', content = EXCLUDED.content, fingerprint = EXCLUDED.fingerprint,
			error_kind = '', error_message = '',
			source = CASE WHEN $11 THEN authoring_segments.source ELSE EXCLUDED.source END,
			duration_ms = CASE WHEN $11 THEN authoring_segments.duration_ms ELSE EXCLUDED.duration_ms END,
			updated_at = now()
	`, projectID, step, s.Key, s.Position, s.Kind, shots, s.Source, s.Fingerprint, []byte(s.Content), s.DurationMS, repaired)
	return err
}

// SaveSegmentFailed marks a segment failed. Its last content, if any, is kept:
// a failed re-run must not throw away what was there.
func (r *PromptTemplateRepository) SaveSegmentFailed(ctx context.Context, projectID, step, key, kind, message string, durationMS int) error {
	return r.updateSegment(ctx, `UPDATE authoring_segments
		SET status = 'failed', error_kind = $4, error_message = $5, duration_ms = $6, updated_at = now()
		WHERE project_id = $1 AND step = $2 AND key = $3`, projectID, step, key, kind, message, durationMS)
}

func (r *PromptTemplateRepository) updateSegment(ctx context.Context, sql string, args ...any) error {
	tag, err := r.pool.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("segment not found")
	}
	return nil
}

// FailRunningSegments ends one project's running segments with a reason, for
// a run that stopped under them (NFR-3). Returns how many there were.
func (r *PromptTemplateRepository) FailRunningSegments(ctx context.Context, projectID, step, kind, message string) (int64, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE authoring_segments
		SET status = 'failed', error_kind = $3, error_message = $4, updated_at = now()
		WHERE project_id = $1 AND step = $2 AND status = 'running'`, projectID, step, kind, message)
	return tag.RowsAffected(), err
}

// FailAllRunningSegments is the startup sweep: nothing can be running when
// this service has just started, so any 'running' row was cut off.
func (r *PromptTemplateRepository) FailAllRunningSegments(ctx context.Context, kind, message string) (int64, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE authoring_segments
		SET status = 'failed', error_kind = $1, error_message = $2, updated_at = now()
		WHERE status = 'running'`, kind, message)
	return tag.RowsAffected(), err
}

func (r *PromptTemplateRepository) DeleteSegments(ctx context.Context, projectID, step string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM authoring_segments WHERE project_id = $1 AND step = $2`, projectID, step)
	return err
}

// GetCodeChunkShots is the Creator's shots per code segment; a project
// without an authoring row yet has the default.
func (r *PromptTemplateRepository) GetCodeChunkShots(ctx context.Context, projectID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT code_chunk_shots FROM project_authoring WHERE project_id = $1`, projectID).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.DefaultChunkShots, nil
	}
	return n, err
}

func (r *PromptTemplateRepository) SaveCodeChunkShots(ctx context.Context, projectID string, n int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO project_authoring (project_id, code_chunk_shots) VALUES ($1, $2)
		ON CONFLICT (project_id) DO UPDATE SET code_chunk_shots = EXCLUDED.code_chunk_shots, updated_at = now()
	`, projectID, n)
	return err
}

// InsertCheckDiagnostics logs one failed check's findings.
func (r *PromptTemplateRepository) InsertCheckDiagnostics(ctx context.Context, list []domain.CheckDiagnosticRecord) error {
	if len(list) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, d := range list {
		var line any
		if d.Line > 0 {
			line = d.Line
		}
		batch.Queue(`INSERT INTO code_check_diagnostics
			(project_id, engine, phase, round, segment_key, shot_id, kind, rule, message, line)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			d.ProjectID, d.Engine, d.Phase, d.Round, d.SegmentKey, d.ShotID, d.Kind, d.Rule, d.Message, line)
	}
	return r.pool.SendBatch(ctx, batch).Close()
}
