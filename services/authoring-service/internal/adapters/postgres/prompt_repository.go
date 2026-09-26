package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"authoring/internal/application"
	"authoring/internal/domain"
)

const promptColumns = `id, role, name, template_text, is_system, is_active, created_at, updated_at`

func scanPrompt(row pgx.Row) (domain.Prompt, error) {
	var p domain.Prompt
	var created, updated time.Time
	if err := row.Scan(&p.ID, &p.Role, &p.Name, &p.TemplateText, &p.IsSystem, &p.IsActive, &created, &updated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Prompt{}, application.ErrPromptNotFound
		}
		return domain.Prompt{}, err
	}
	p.CreatedAt = created.Format(time.RFC3339)
	p.UpdatedAt = updated.Format(time.RFC3339)
	return p, nil
}

// PurgeLegacyPrompts removes what the pre-library override system left
// behind: the "migrated-*" Creator rows it produced and the renamed *_legacy
// tables. Idempotent — with nothing left it is a no-op.
func (r *PromptTemplateRepository) PurgeLegacyPrompts(ctx context.Context) (int, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM prompts WHERE id LIKE 'migrated-%' AND NOT is_system`)
	if err != nil {
		return 0, err
	}
	for _, t := range []string{"prompt_overrides_legacy", "prompt_templates_legacy", "prompt_overrides", "prompt_templates"} {
		if _, err := r.pool.Exec(ctx, `DROP TABLE IF EXISTS `+t); err != nil {
			return int(tag.RowsAffected()), err
		}
	}
	return int(tag.RowsAffected()), nil
}

// SeedPrompts writes the system prompts, overwriting their wording on every
// start so a rebuilt binary is the whole deploy.
//
// A role's system row is made active only when the role has no active row yet
// — the first run — so a restart never undoes the Creator's choice.
func (r *PromptTemplateRepository) SeedPrompts(ctx context.Context) error {
	for _, p := range domain.SystemPrompts() {
		if _, err := r.pool.Exec(ctx, `
			INSERT INTO prompts (id, role, name, template_text, is_system, is_active, updated_at)
			VALUES ($1, $2, $3, $4, true,
			        NOT EXISTS (SELECT 1 FROM prompts WHERE role = $2 AND is_active),
			        now())
			ON CONFLICT (id) DO UPDATE SET
			    name = EXCLUDED.name,
			    template_text = EXCLUDED.template_text,
			    updated_at = now()
		`, p.ID, string(p.Role), p.Name, p.TemplateText); err != nil {
			return err
		}
	}
	return nil
}

// GetActive returns the row a role runs on: the active one, and if none is
// (a role whose active row was just deleted) its system row.
func (r *PromptTemplateRepository) GetActive(ctx context.Context, role domain.PromptRole) (domain.Prompt, error) {
	return scanPrompt(r.pool.QueryRow(ctx, `
		SELECT `+promptColumns+` FROM prompts WHERE role = $1
		ORDER BY is_active DESC, is_system DESC LIMIT 1
	`, string(role)))
}

func (r *PromptTemplateRepository) GetSystem(ctx context.Context, role domain.PromptRole) (domain.Prompt, error) {
	return scanPrompt(r.pool.QueryRow(ctx,
		`SELECT `+promptColumns+` FROM prompts WHERE role = $1 AND is_system`, string(role)))
}

func (r *PromptTemplateRepository) Get(ctx context.Context, id string) (domain.Prompt, error) {
	return scanPrompt(r.pool.QueryRow(ctx, `SELECT `+promptColumns+` FROM prompts WHERE id = $1`, id))
}

// List returns the library grouped by role, system row first, then the
// Creator's rows oldest first.
func (r *PromptTemplateRepository) List(ctx context.Context, role domain.PromptRole) ([]domain.Prompt, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+promptColumns+` FROM prompts
		WHERE ($1 = '' OR role = $1)
		ORDER BY role, is_system DESC, created_at, id
	`, string(role))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Prompt, 0)
	for rows.Next() {
		p, err := scanPrompt(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PromptTemplateRepository) Create(ctx context.Context, role domain.PromptRole, name, templateText string) (domain.Prompt, error) {
	return scanPrompt(r.pool.QueryRow(ctx, `
		INSERT INTO prompts (role, name, template_text) VALUES ($1, $2, $3)
		RETURNING `+promptColumns, string(role), name, templateText))
}

// Update refuses system rows in SQL as well, so a caller that skipped the
// use case's check still cannot edit one.
func (r *PromptTemplateRepository) Update(ctx context.Context, id, name, templateText string) (domain.Prompt, error) {
	return scanPrompt(r.pool.QueryRow(ctx, `
		UPDATE prompts SET name = $2, template_text = $3, updated_at = now()
		WHERE id = $1 AND NOT is_system
		RETURNING `+promptColumns, id, name, templateText))
}

func (r *PromptTemplateRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM prompts WHERE id = $1 AND NOT is_system`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return application.ErrPromptNotFound
	}
	return nil
}

// Activate switches a role's active row in one transaction: the old one off
// first, because the unique index would refuse two at once.
func (r *PromptTemplateRepository) Activate(ctx context.Context, id string) (domain.Prompt, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Prompt{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var role string
	if err := tx.QueryRow(ctx, `SELECT role FROM prompts WHERE id = $1 FOR UPDATE`, id).Scan(&role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Prompt{}, application.ErrPromptNotFound
		}
		return domain.Prompt{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE prompts SET is_active = false WHERE role = $1 AND is_active AND id <> $2`, role, id); err != nil {
		return domain.Prompt{}, err
	}
	p, err := scanPrompt(tx.QueryRow(ctx, `
		UPDATE prompts SET is_active = true WHERE id = $1 RETURNING `+promptColumns, id))
	if err != nil {
		return domain.Prompt{}, fmt.Errorf("activate: %w", err)
	}
	return p, tx.Commit(ctx)
}
