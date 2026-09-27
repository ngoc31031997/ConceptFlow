package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"authoring/internal/domain"
)

// CR-044 — the drawings one video needs, joined with the library row they
// point at so the review screen and the code-step gate read one list.

func (r *PromptTemplateRepository) ListProjectIllustrations(ctx context.Context, projectID string) ([]domain.ProjectIllustration, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, position, name, description, folder_id, shots, state, error, COALESCE(illustration_id, '')
		FROM project_illustrations WHERE project_id = $1 ORDER BY position
	`, projectID)
	if err != nil {
		return nil, err
	}
	var out []domain.ProjectIllustration
	for rows.Next() {
		var p domain.ProjectIllustration
		var state string
		if err := rows.Scan(&p.ID, &p.ProjectID, &p.Position, &p.Name, &p.Description, &p.FolderID, &p.Shots,
			&state, &p.Error, &p.IllustrationID); err != nil {
			rows.Close()
			return nil, err
		}
		p.State = domain.ProjectIllustrationState(state)
		if p.Shots == nil {
			p.Shots = []string{}
		}
		out = append(out, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].IllustrationID == "" {
			continue
		}
		ill, err := r.GetIllustration(ctx, out[i].IllustrationID)
		if err == nil {
			out[i].Illustration = &ill
		}
	}
	if out == nil {
		out = []domain.ProjectIllustration{}
	}
	return out, nil
}

func (r *PromptTemplateRepository) ReplaceProjectIllustrations(ctx context.Context, projectID string, list []domain.ProjectIllustration) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM project_illustrations WHERE project_id = $1`, projectID); err != nil {
			return err
		}
		for _, p := range list {
			var ill any
			if p.IllustrationID != "" {
				ill = p.IllustrationID
			}
			shots := p.Shots
			if shots == nil {
				shots = []string{}
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO project_illustrations (project_id, position, name, description, folder_id, shots, state, error, illustration_id)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			`, projectID, p.Position, p.Name, p.Description, p.FolderID, shots, string(p.State), p.Error, ill); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PromptTemplateRepository) UpdateProjectIllustration(ctx context.Context, p domain.ProjectIllustration) error {
	var ill any
	if p.IllustrationID != "" {
		ill = p.IllustrationID
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE project_illustrations SET name = $3, state = $4, error = $5, illustration_id = $6
		WHERE id = $1 AND project_id = $2
	`, p.ID, p.ProjectID, p.Name, string(p.State), p.Error, ill)
	return err
}

// DeleteProjectIllustrations drops a deleted project's list (CR-040 FR114 cleanup).
func (r *PromptTemplateRepository) DeleteProjectIllustrations(ctx context.Context, projectID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM project_illustrations WHERE project_id = $1`, projectID)
	return err
}

// MarkIllustrationsPlanned records that the video's drawing list was made (CR-045).
func (r *PromptTemplateRepository) MarkIllustrationsPlanned(ctx context.Context, projectID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO project_authoring (project_id, illustrations_planned_at) VALUES ($1, now())
		ON CONFLICT (project_id) DO UPDATE SET illustrations_planned_at = now()
	`, projectID)
	return err
}

// IllustrationsPlanned reports whether the video's drawing list was ever made.
func (r *PromptTemplateRepository) IllustrationsPlanned(ctx context.Context, projectID string) (bool, error) {
	var planned bool
	err := r.pool.QueryRow(ctx, `
		SELECT illustrations_planned_at IS NOT NULL FROM project_authoring WHERE project_id = $1
	`, projectID).Scan(&planned)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return planned, err
}
