package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"authoring/internal/application"
	"authoring/internal/domain"
)

const archetypeColumns = `id, code, name, when_to_use, playbook, is_system, created_at, updated_at`

func scanArchetype(row pgx.Row) (domain.VideoArchetype, error) {
	var a domain.VideoArchetype
	var created, updated time.Time
	if err := row.Scan(&a.ID, &a.Code, &a.Name, &a.WhenToUse, &a.Playbook, &a.IsSystem, &created, &updated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.VideoArchetype{}, application.ErrArchetypeNotFound
		}
		return domain.VideoArchetype{}, err
	}
	a.CreatedAt = created.Format(time.RFC3339)
	a.UpdatedAt = updated.Format(time.RFC3339)
	return a, nil
}

// mapArchetypeErr turns the unique-code violation into a domain error.
func mapArchetypeErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return application.ErrArchetypeCodeTaken
	}
	return err
}

// SeedVideoArchetypes refreshes the system rows' wording on every start, like
// SeedPrompts. A Creator row that already took a system code is left alone and
// the system row is skipped for this start rather than failing boot.
func (r *PromptTemplateRepository) SeedVideoArchetypes(ctx context.Context) error {
	for _, a := range domain.SystemVideoArchetypes() {
		if _, err := r.pool.Exec(ctx, `
			INSERT INTO video_archetypes (id, code, name, when_to_use, playbook, is_system, updated_at)
			VALUES ($1, $2, $3, $4, $5, true, now())
			ON CONFLICT (id) DO UPDATE SET
			    code = EXCLUDED.code, name = EXCLUDED.name,
			    when_to_use = EXCLUDED.when_to_use, playbook = EXCLUDED.playbook,
			    updated_at = now()
		`, a.ID, a.Code, a.Name, a.WhenToUse, a.Playbook); err != nil {
			if errors.Is(mapArchetypeErr(err), application.ErrArchetypeCodeTaken) {
				continue
			}
			return err
		}
	}
	return nil
}

// ListArchetypes returns system rows first, then the Creator's, oldest first.
func (r *PromptTemplateRepository) ListArchetypes(ctx context.Context) ([]domain.VideoArchetype, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+archetypeColumns+` FROM video_archetypes
		ORDER BY is_system DESC, upper(code), created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.VideoArchetype, 0)
	for rows.Next() {
		a, err := scanArchetype(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *PromptTemplateRepository) GetArchetype(ctx context.Context, id string) (domain.VideoArchetype, error) {
	return scanArchetype(r.pool.QueryRow(ctx, `SELECT `+archetypeColumns+` FROM video_archetypes WHERE id = $1`, id))
}

func (r *PromptTemplateRepository) CreateArchetype(ctx context.Context, a domain.VideoArchetype) (domain.VideoArchetype, error) {
	out, err := scanArchetype(r.pool.QueryRow(ctx, `
		INSERT INTO video_archetypes (code, name, when_to_use, playbook) VALUES ($1, $2, $3, $4)
		RETURNING `+archetypeColumns, a.Code, a.Name, a.WhenToUse, a.Playbook))
	return out, mapArchetypeErr(err)
}

// UpdateArchetype refuses system rows in SQL as well as in the use case.
func (r *PromptTemplateRepository) UpdateArchetype(ctx context.Context, a domain.VideoArchetype) (domain.VideoArchetype, error) {
	out, err := scanArchetype(r.pool.QueryRow(ctx, `
		UPDATE video_archetypes SET code = $2, name = $3, when_to_use = $4, playbook = $5, updated_at = now()
		WHERE id = $1 AND NOT is_system
		RETURNING `+archetypeColumns, a.ID, a.Code, a.Name, a.WhenToUse, a.Playbook))
	return out, mapArchetypeErr(err)
}

func (r *PromptTemplateRepository) DeleteArchetype(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM video_archetypes WHERE id = $1 AND NOT is_system`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return application.ErrArchetypeNotFound
	}
	return nil
}
