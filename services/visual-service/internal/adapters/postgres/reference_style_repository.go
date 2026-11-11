package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"visual/internal/application"
	"visual/internal/domain"
)

const referenceStyleColumns = "id, name, description, tags, detail, created_at, updated_at"

// referenceDetail is the JSON stored in reference_styles.detail.
type referenceDetail struct {
	Source  domain.ReferenceSource  `json:"source"`
	Drawing domain.ReferenceDrawing `json:"drawing"`
	Layout  domain.ReferenceLayout  `json:"layout"`
}

func detailJSON(in domain.ReferenceStyleInput) ([]byte, error) {
	return json.Marshal(referenceDetail{Source: in.Source, Drawing: in.Drawing, Layout: in.Layout})
}

// ReferenceStyleRepository implements application.ReferenceStylePort.
type ReferenceStyleRepository struct {
	pool *pgxpool.Pool
}

// NewReferenceStyleRepository returns the repository over pool.
func NewReferenceStyleRepository(pool *pgxpool.Pool) *ReferenceStyleRepository {
	return &ReferenceStyleRepository{pool: pool}
}

func scanReferenceStyle(row pgx.Row) (domain.ReferenceStyle, error) {
	var s domain.ReferenceStyle
	var detail []byte
	if err := row.Scan(&s.ID, &s.Name, &s.Summary, &s.Tags, &detail, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return s, err
	}
	var d referenceDetail
	if err := json.Unmarshal(detail, &d); err != nil {
		return s, fmt.Errorf("decode reference style detail: %w", err)
	}
	s.Source, s.Drawing, s.Layout = d.Source, d.Drawing, d.Layout
	if s.Tags == nil {
		s.Tags = []string{}
	}
	return s, nil
}

// ListReferenceStyles returns every description ordered by seq.
func (r *ReferenceStyleRepository) ListReferenceStyles(ctx context.Context) ([]domain.ReferenceStyle, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+referenceStyleColumns+" FROM reference_styles ORDER BY seq")
	if err != nil {
		return nil, fmt.Errorf("list reference styles: %w", err)
	}
	defer rows.Close()
	styles := []domain.ReferenceStyle{}
	for rows.Next() {
		s, err := scanReferenceStyle(rows)
		if err != nil {
			return nil, fmt.Errorf("scan reference style: %w", err)
		}
		styles = append(styles, s)
	}
	return styles, rows.Err()
}

// CreateReferenceStyle takes the next number from reference_styles_seq, so a
// deleted id is never handed out again.
func (r *ReferenceStyleRepository) CreateReferenceStyle(ctx context.Context, in domain.ReferenceStyleInput) (domain.ReferenceStyle, error) {
	var seq int64
	if err := r.pool.QueryRow(ctx, "SELECT nextval('reference_styles_seq')").Scan(&seq); err != nil {
		return domain.ReferenceStyle{}, fmt.Errorf("next reference style id: %w", err)
	}
	detail, err := detailJSON(in)
	if err != nil {
		return domain.ReferenceStyle{}, fmt.Errorf("encode reference style detail: %w", err)
	}
	s, err := scanReferenceStyle(r.pool.QueryRow(ctx,
		`INSERT INTO reference_styles (id, seq, name, description, tags, detail) VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+referenceStyleColumns,
		domain.ReferenceStyleID(seq), seq, in.Name, in.Summary, in.Tags, detail))
	if err != nil {
		return domain.ReferenceStyle{}, fmt.Errorf("insert reference style: %w", err)
	}
	return s, nil
}

// UpdateReferenceStyle replaces every field of id.
func (r *ReferenceStyleRepository) UpdateReferenceStyle(ctx context.Context, id string, in domain.ReferenceStyleInput) (domain.ReferenceStyle, error) {
	detail, err := detailJSON(in)
	if err != nil {
		return domain.ReferenceStyle{}, fmt.Errorf("encode reference style detail: %w", err)
	}
	s, err := scanReferenceStyle(r.pool.QueryRow(ctx,
		`UPDATE reference_styles SET name = $2, description = $3, tags = $4, detail = $5, updated_at = now()
		 WHERE id = $1 RETURNING `+referenceStyleColumns,
		id, in.Name, in.Summary, in.Tags, detail))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ReferenceStyle{}, application.ErrReferenceStyleNotFound
	}
	if err != nil {
		return domain.ReferenceStyle{}, fmt.Errorf("update reference style: %w", err)
	}
	return s, nil
}

// DeleteReferenceStyle removes id; the foreign key removes the videos' picks.
func (r *ReferenceStyleRepository) DeleteReferenceStyle(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM reference_styles WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete reference style: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return application.ErrReferenceStyleNotFound
	}
	return nil
}

// ReferenceStyleUsers returns the projects that picked id, ordered by id.
func (r *ReferenceStyleRepository) ReferenceStyleUsers(ctx context.Context, id string) ([]string, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM reference_styles WHERE id = $1)", id).Scan(&exists); err != nil {
		return nil, fmt.Errorf("find reference style: %w", err)
	}
	if !exists {
		return nil, application.ErrReferenceStyleNotFound
	}
	rows, err := r.pool.Query(ctx, "SELECT project_id FROM project_reference_picks WHERE ref_id = $1 ORDER BY project_id", id)
	if err != nil {
		return nil, fmt.Errorf("list reference style users: %w", err)
	}
	defer rows.Close()
	projects := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("scan reference style user: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}
