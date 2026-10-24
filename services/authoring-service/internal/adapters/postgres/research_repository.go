package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// interruptedCheckWhy is what a link check left running when the service
// stopped reads as after the next start.
const interruptedCheckWhy = "bị gián đoạn khi dịch vụ khởi động lại, bấm Kiểm lại"

const factColumns = `fact_id, question_id, claim_vi, type, source_title, publisher, year, url, quote,
	confidence, note, must, keep, link_state, link_why, link_nearest, link_checked_at, creator_fields`

type factScanner interface{ Scan(dest ...any) error }

func scanFact(row factScanner) (domain.StoredFact, error) {
	var s domain.StoredFact
	var typ, state string
	var creator []byte
	f := &s.Fact
	if err := row.Scan(&s.ID, &f.Question, &f.ClaimVI, &typ, &f.SourceTitle, &f.Publisher, &f.Year, &f.URL, &f.Quote,
		&f.Confidence, &f.Note, &f.Must, &f.Keep, &state, &f.Link.Why, &f.Link.Nearest, &f.Link.CheckedAt, &creator); err != nil {
		return domain.StoredFact{}, err
	}
	f.Type, f.Link.State = domain.FactType(typ), domain.LinkState(state)
	if err := json.Unmarshal(creator, &s.CreatorFields); err != nil {
		return domain.StoredFact{}, fmt.Errorf("read creator fields of fact %s: %w", s.ID, err)
	}
	if s.CreatorFields == nil {
		s.CreatorFields = []string{}
	}
	return s, nil
}

// ListResearchFacts returns the project's facts in code order (F2 before F10).
func (r *PromptTemplateRepository) ListResearchFacts(ctx context.Context, projectID string) ([]domain.StoredFact, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+factColumns+` FROM project_research
		WHERE project_id = $1 ORDER BY length(fact_id), fact_id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	facts := []domain.StoredFact{}
	for rows.Next() {
		s, err := scanFact(rows)
		if err != nil {
			return nil, err
		}
		facts = append(facts, s)
	}
	return facts, rows.Err()
}

// GetResearchFact returns one fact, or domain.ErrUnitNotFound.
func (r *PromptTemplateRepository) GetResearchFact(ctx context.Context, projectID, factID string) (domain.StoredFact, error) {
	s, err := scanFact(r.pool.QueryRow(ctx, `SELECT `+factColumns+` FROM project_research
		WHERE project_id = $1 AND fact_id = $2`, projectID, factID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.StoredFact{}, domain.ErrUnitNotFound
	}
	return s, err
}

// SaveResearchFact inserts a fact or replaces the one with the same code.
func (r *PromptTemplateRepository) SaveResearchFact(ctx context.Context, projectID string, s domain.StoredFact) error {
	creator := s.CreatorFields
	if creator == nil {
		creator = []string{}
	}
	raw, err := json.Marshal(creator)
	if err != nil {
		return err
	}
	f := s.Fact
	_, err = r.pool.Exec(ctx, `
		INSERT INTO project_research (project_id, `+factColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19::jsonb)
		ON CONFLICT (project_id, fact_id) DO UPDATE SET
		    question_id = EXCLUDED.question_id, claim_vi = EXCLUDED.claim_vi, type = EXCLUDED.type,
		    source_title = EXCLUDED.source_title, publisher = EXCLUDED.publisher, year = EXCLUDED.year,
		    url = EXCLUDED.url, quote = EXCLUDED.quote, confidence = EXCLUDED.confidence, note = EXCLUDED.note,
		    must = EXCLUDED.must, keep = EXCLUDED.keep, link_state = EXCLUDED.link_state, link_why = EXCLUDED.link_why,
		    link_nearest = EXCLUDED.link_nearest, link_checked_at = EXCLUDED.link_checked_at,
		    creator_fields = EXCLUDED.creator_fields, updated_at = now()
	`, projectID, s.ID, f.Question, f.ClaimVI, string(f.Type), f.SourceTitle, f.Publisher, f.Year, f.URL, f.Quote,
		f.Confidence, f.Note, f.Must, f.Keep, string(f.Link.State), f.Link.Why, f.Link.Nearest, f.Link.CheckedAt, string(raw))
	return err
}

// SaveFactLink stores the result of checking a fact's link, but only while the
// fact still has the url and quote the check started from: a Creator who edited
// them meanwhile keeps the newer fact and the stale result is dropped. It
// reports whether the result was stored.
func (r *PromptTemplateRepository) SaveFactLink(ctx context.Context, projectID, key, expectedURL, expectedQuote string, link domain.FactLink) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE project_research
		SET link_state = $4, link_why = $5, link_nearest = $6, link_checked_at = $7, updated_at = now()
		WHERE project_id = $1 AND fact_id = $2 AND url = $3 AND quote = $8
	`, projectID, key, expectedURL, string(link.State), link.Why, link.Nearest, link.CheckedAt, expectedQuote)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// ResetRunningFactLinks turns every link check still marked running into an
// unreachable one with an explanation: no check survives a restart, so a
// running mark found at start-up is a leftover. It returns how many it fixed.
func (r *PromptTemplateRepository) ResetRunningFactLinks(ctx context.Context) (int, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE project_research
		SET link_state = $1, link_why = $2, link_nearest = '', link_checked_at = '', updated_at = now()
		WHERE link_state = 'run'
	`, string(domain.LinkUnreachable), interruptedCheckWhy)
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
