package postgres

import (
	"context"
	"encoding/json"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// interruptedCheckWhy is what a link check left running when the service
// stopped reads as after the next start.
const interruptedCheckWhy = "bị gián đoạn khi dịch vụ khởi động lại, bấm Kiểm lại"

// SaveFactLink stores the result of checking a fact's link, but only while the
// fact still has the url and quote the check started from: a Creator who edited
// them meanwhile keeps the newer fact and the stale result is dropped. It
// reports whether the result was stored.
func (r *PromptTemplateRepository) SaveFactLink(ctx context.Context, projectID, key, expectedURL, expectedQuote string, link domain.FactLink) (bool, error) {
	raw, err := json.Marshal(link)
	if err != nil {
		return false, err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE review_units
		SET content = jsonb_set(content, '{link}', $4::jsonb), updated_at = now()
		WHERE project_id = $1 AND kind = 'fact' AND unit_key = $2
		  AND content->>'url' = $3 AND content->>'quote' = $5
	`, projectID, key, expectedURL, string(raw), expectedQuote)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// ResetRunningFactLinks turns every link check still marked running into an
// unreachable one with an explanation: no check survives a restart, so a
// running mark found at start-up is a leftover. It returns how many it fixed.
func (r *PromptTemplateRepository) ResetRunningFactLinks(ctx context.Context) (int, error) {
	raw, err := json.Marshal(domain.FactLink{State: domain.LinkUnreachable, Why: interruptedCheckWhy})
	if err != nil {
		return 0, err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE review_units
		SET content = jsonb_set(content, '{link}', $1::jsonb), updated_at = now()
		WHERE kind = 'fact' AND content->'link'->>'state' = 'run'
	`, string(raw))
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// AddPaste records a pasted answer as the project's next numbered paste and
// returns it with its number and time.
func (r *PromptTemplateRepository) AddPaste(ctx context.Context, projectID string, p application.PasteLog) (application.PasteLog, error) {
	rejected := p.Rejected
	if rejected == nil {
		rejected = []domain.PasteRejected{}
	}
	raw, err := json.Marshal(rejected)
	if err != nil {
		return application.PasteLog{}, err
	}
	added, updated := p.Added, p.Updated
	if added == nil {
		added = []string{}
	}
	if updated == nil {
		updated = []string{}
	}
	out := application.PasteLog{Added: added, Updated: updated, Rejected: rejected}
	err = r.pool.QueryRow(ctx, `
		INSERT INTO research_pastes (project_id, n, added, updated, rejected)
		VALUES ($1, COALESCE((SELECT max(n) FROM research_pastes WHERE project_id = $1), 0) + 1, $2, $3, $4::jsonb)
		RETURNING n, at
	`, projectID, added, updated, string(raw)).Scan(&out.N, &out.At)
	return out, err
}

// ListPastes returns the project's pasted answers, oldest first.
func (r *PromptTemplateRepository) ListPastes(ctx context.Context, projectID string) ([]application.PasteLog, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT n, at, added, updated, rejected FROM research_pastes WHERE project_id = $1 ORDER BY n
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	logs := []application.PasteLog{}
	for rows.Next() {
		var p application.PasteLog
		var rejected []byte
		if err := rows.Scan(&p.N, &p.At, &p.Added, &p.Updated, &rejected); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(rejected, &p.Rejected); err != nil {
			return nil, err
		}
		if p.Added == nil {
			p.Added = []string{}
		}
		if p.Updated == nil {
			p.Updated = []string{}
		}
		if p.Rejected == nil {
			p.Rejected = []domain.PasteRejected{}
		}
		logs = append(logs, p)
	}
	return logs, rows.Err()
}
