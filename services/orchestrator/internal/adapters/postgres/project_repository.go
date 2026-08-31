package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"orchestrator/internal/domain"
)

// ProjectRepository implements domain.ProjectRepositoryPort against the
// projects/saga_steps tables (CRUD, not CQRS — nfr-design-patterns.md).
type ProjectRepository struct {
	pool *pgxpool.Pool
}

// NewProjectRepository constructs the repository over an already-open pool.
func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{pool: pool}
}

// Get loads a Project by project_id, or domain.ErrProjectNotFound.
func (r *ProjectRepository) Get(ctx context.Context, projectID string) (*domain.Project, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT project_id, saga_id, status, script_content, plugin_id, voice_language,
		       background_music_path, scenes, video_path, youtube_title, youtube_description,
		       youtube_tags, youtube_visibility, youtube_video_url, error_message
		FROM projects WHERE project_id = $1`, projectID)

	var (
		p                     domain.Project
		status, voiceLanguage string
		scenesJSON            []byte
		tagsJSON              []byte
		youtubeVisibility     *string
	)
	err := row.Scan(&p.ProjectID, &p.SagaID, &status, &p.ScriptContent, &p.PluginID, &voiceLanguage,
		&p.BackgroundMusicPath, &scenesJSON, &p.VideoPath, &p.YoutubeTitle, &p.YoutubeDescription,
		&tagsJSON, &youtubeVisibility, &p.YoutubeVideoURL, &p.ErrorMessage)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrProjectNotFound
	}
	if err != nil {
		return nil, err
	}

	p.Status = domain.ProjectStatus(status)
	p.VoiceLanguage = domain.VoiceLanguage(voiceLanguage)
	if youtubeVisibility != nil {
		v := domain.Visibility(*youtubeVisibility)
		p.YoutubeVisibility = &v
	}
	if len(scenesJSON) > 0 {
		if err := json.Unmarshal(scenesJSON, &p.Scenes); err != nil {
			return nil, err
		}
	}
	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &p.YoutubeTags); err != nil {
			return nil, err
		}
	}
	return &p, nil
}

// Save upserts a Project (CRUD — module-structure.md).
func (r *ProjectRepository) Save(ctx context.Context, project *domain.Project) error {
	scenesJSON, err := json.Marshal(project.Scenes)
	if err != nil {
		return err
	}
	tagsJSON, err := json.Marshal(project.YoutubeTags)
	if err != nil {
		return err
	}
	var youtubeVisibility *string
	if project.YoutubeVisibility != nil {
		v := string(*project.YoutubeVisibility)
		youtubeVisibility = &v
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO projects (project_id, saga_id, status, script_content, plugin_id, voice_language,
		                       background_music_path, scenes, video_path, youtube_title, youtube_description,
		                       youtube_tags, youtube_visibility, youtube_video_url, error_message, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15, now())
		ON CONFLICT (project_id) DO UPDATE SET
		    saga_id = EXCLUDED.saga_id, status = EXCLUDED.status, script_content = EXCLUDED.script_content,
		    plugin_id = EXCLUDED.plugin_id, voice_language = EXCLUDED.voice_language,
		    background_music_path = EXCLUDED.background_music_path, scenes = EXCLUDED.scenes,
		    video_path = EXCLUDED.video_path, youtube_title = EXCLUDED.youtube_title,
		    youtube_description = EXCLUDED.youtube_description, youtube_tags = EXCLUDED.youtube_tags,
		    youtube_visibility = EXCLUDED.youtube_visibility, youtube_video_url = EXCLUDED.youtube_video_url,
		    error_message = EXCLUDED.error_message, updated_at = now()`,
		project.ProjectID, project.SagaID, string(project.Status), project.ScriptContent, project.PluginID,
		string(project.VoiceLanguage), project.BackgroundMusicPath, scenesJSON, project.VideoPath,
		project.YoutubeTitle, project.YoutubeDescription, tagsJSON, youtubeVisibility,
		project.YoutubeVideoURL, project.ErrorMessage)
	return err
}

// UpdateStatus updates only Project.Status.
func (r *ProjectRepository) UpdateStatus(ctx context.Context, projectID string, status domain.ProjectStatus) error {
	tag, err := r.pool.Exec(ctx, `UPDATE projects SET status = $1, updated_at = now() WHERE project_id = $2`, string(status), projectID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrProjectNotFound
	}
	return nil
}

// GetStep loads a SagaStep by (saga_id, step_name), or domain.ErrSagaStepNotFound.
func (r *ProjectRepository) GetStep(ctx context.Context, sagaID string, stepName domain.StepName) (*domain.SagaStep, error) {
	row := r.pool.QueryRow(ctx, `SELECT saga_id, step_name, status, error_message FROM saga_steps WHERE saga_id = $1 AND step_name = $2`,
		sagaID, string(stepName))

	var s domain.SagaStep
	var step, status string
	err := row.Scan(&s.SagaID, &step, &status, &s.ErrorMessage)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSagaStepNotFound
	}
	if err != nil {
		return nil, err
	}
	s.StepName = domain.StepName(step)
	s.Status = domain.SagaStepStatus(status)
	return &s, nil
}

// UpdateStep upserts a SagaStep.
func (r *ProjectRepository) UpdateStep(ctx context.Context, step *domain.SagaStep) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO saga_steps (saga_id, step_name, status, error_message, updated_at)
		VALUES ($1,$2,$3,$4, now())
		ON CONFLICT (saga_id, step_name) DO UPDATE SET
		    status = EXCLUDED.status, error_message = EXCLUDED.error_message, updated_at = now()`,
		step.SagaID, string(step.StepName), string(step.Status), step.ErrorMessage)
	return err
}
