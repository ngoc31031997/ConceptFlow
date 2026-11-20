package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"visual/internal/application"
)

// StoryboardRepository implements application.StoryboardStorePort. Its review
// units are the shared review_units table; the locked concept, plan and sequence
// it reads come through the sequence repository. The locked storyboards are
// kept by rendering, not here.
type StoryboardRepository struct {
	*SequenceRepository
}

var _ application.StoryboardStorePort = (*StoryboardRepository)(nil)

// NewStoryboardRepository returns the repository over pool.
func NewStoryboardRepository(pool *pgxpool.Pool) *StoryboardRepository {
	return &StoryboardRepository{SequenceRepository: NewSequenceRepository(pool)}
}

// StoryboardModel returns the video's model of step 11; "" when none is saved.
func (r *StoryboardRepository) StoryboardModel(ctx context.Context, projectID string) (string, error) {
	var m string
	err := r.pool.QueryRow(ctx, `SELECT storyboard_model FROM project_visual WHERE project_id = $1`, projectID).Scan(&m)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read storyboard model: %w", err)
	}
	return m, nil
}

// SetStoryboardModel saves the video's model of step 11.
func (r *StoryboardRepository) SetStoryboardModel(ctx context.Context, projectID, model string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO project_visual (project_id, storyboard_model) VALUES ($1, $2)
		ON CONFLICT (project_id) DO UPDATE SET storyboard_model = EXCLUDED.storyboard_model, updated_at = now()`, projectID, model)
	if err != nil {
		return fmt.Errorf("save storyboard model: %w", err)
	}
	return nil
}
