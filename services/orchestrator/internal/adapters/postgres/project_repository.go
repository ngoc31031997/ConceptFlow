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
		SELECT project_id, saga_id, status, script_content, manim_scene_class_name, plugin_id, category_hint, voice_language,
		       background_music_path, scenes, rendered_video_path, video_path, youtube_title, youtube_description,
		       youtube_tags, youtube_visibility, youtube_publish_at, youtube_thumbnail_path, youtube_video_url, error_message,
		       tts_enabled, voice_id, subtitles_enabled, subtitle_style, wait_offsets, rendered_video_seconds,
		       render_quality, background_music_volume
		FROM projects WHERE project_id = $1`, projectID)

	var (
		p                     domain.Project
		status, voiceLanguage string
		scenesJSON            []byte
		tagsJSON              []byte
		youtubeVisibility     *string
		subtitleStyleJSON     []byte
		waitOffsetsJSON       []byte
		renderQuality         string
	)
	err := row.Scan(&p.ProjectID, &p.SagaID, &status, &p.ScriptContent, &p.ManimSceneClassName, &p.PluginID, &p.CategoryHint, &voiceLanguage,
		&p.BackgroundMusicPath, &scenesJSON, &p.RenderedVideoPath, &p.VideoPath, &p.YoutubeTitle, &p.YoutubeDescription,
		&tagsJSON, &youtubeVisibility, &p.YoutubePublishAt, &p.YoutubeThumbnailPath, &p.YoutubeVideoURL, &p.ErrorMessage,
		&p.TTSEnabled, &p.VoiceID, &p.SubtitlesEnabled, &subtitleStyleJSON, &waitOffsetsJSON, &p.RenderedVideoSeconds,
		&renderQuality, &p.BackgroundMusicVolume)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrProjectNotFound
	}
	if err != nil {
		return nil, err
	}

	p.Status = domain.ProjectStatus(status)
	p.RenderQuality = domain.RenderQuality(renderQuality)
	p.ContentLanguage = domain.ContentLanguage(voiceLanguage)
	if youtubeVisibility != nil {
		v := domain.Visibility(*youtubeVisibility)
		p.YoutubeVisibility = &v
	}
	if len(waitOffsetsJSON) > 0 {
		if err := json.Unmarshal(waitOffsetsJSON, &p.WaitOffsets); err != nil {
			return nil, err
		}
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
	if len(subtitleStyleJSON) > 0 {
		var style domain.SubtitleStyle
		if err := json.Unmarshal(subtitleStyleJSON, &style); err != nil {
			return nil, err
		}
		p.SubtitleStyle = &style
	}
	return &p, nil
}

// List returns every project as a lightweight ProjectSummary (no
// scenes/script_content), newest-updated first, for GET /v1/projects.
func (r *ProjectRepository) List(ctx context.Context) ([]domain.ProjectSummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT project_id, status, video_path, error_message, updated_at
		FROM projects ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaries := make([]domain.ProjectSummary, 0)
	for rows.Next() {
		var s domain.ProjectSummary
		var status string
		if err := rows.Scan(&s.ProjectID, &status, &s.VideoPath, &s.ErrorMessage, &s.UpdatedAt); err != nil {
			return nil, err
		}
		s.Status = domain.ProjectStatus(status)
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

// Delete removes a project and everything derived from it — the projects
// row itself, its saga_steps (matched by saga_id), and any outbox_events
// still queued for it (matched by the project_id embedded in the command
// payload) — so a deleted video leaves no residual rows behind to bloat the
// database. File cleanup on the shared volume is the caller's (Gateway's)
// responsibility, not the Orchestrator's (it has no mount of that volume).
func (r *ProjectRepository) Delete(ctx context.Context, projectID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var sagaID string
	err = tx.QueryRow(ctx, `DELETE FROM projects WHERE project_id = $1 RETURNING saga_id`, projectID).Scan(&sagaID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrProjectNotFound
	}
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM saga_steps WHERE saga_id = $1`, sagaID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM outbox_events WHERE payload->>'project_id' = $1`, projectID); err != nil {
		return err
	}

	return tx.Commit(ctx)
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
	var subtitleStyleJSON []byte
	if project.SubtitleStyle != nil {
		if subtitleStyleJSON, err = json.Marshal(project.SubtitleStyle); err != nil {
			return err
		}
	}

	var waitOffsetsJSON []byte
	if project.WaitOffsets != nil {
		if waitOffsetsJSON, err = json.Marshal(project.WaitOffsets); err != nil {
			return err
		}
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO projects (project_id, saga_id, status, script_content, manim_scene_class_name, plugin_id, category_hint, voice_language,
		                       background_music_path, scenes, rendered_video_path, video_path, youtube_title, youtube_description,
		                       youtube_tags, youtube_visibility, youtube_publish_at, youtube_thumbnail_path, youtube_video_url, error_message,
		                       tts_enabled, voice_id, subtitles_enabled, subtitle_style, wait_offsets, rendered_video_seconds,
		                       render_quality, background_music_volume, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28, now())
		ON CONFLICT (project_id) DO UPDATE SET
		    saga_id = EXCLUDED.saga_id, status = EXCLUDED.status, script_content = EXCLUDED.script_content,
		    manim_scene_class_name = EXCLUDED.manim_scene_class_name,
		    plugin_id = EXCLUDED.plugin_id, category_hint = EXCLUDED.category_hint, voice_language = EXCLUDED.voice_language,
		    background_music_path = EXCLUDED.background_music_path, scenes = EXCLUDED.scenes,
		    rendered_video_path = EXCLUDED.rendered_video_path,
		    video_path = EXCLUDED.video_path, youtube_title = EXCLUDED.youtube_title,
		    youtube_description = EXCLUDED.youtube_description, youtube_tags = EXCLUDED.youtube_tags,
		    youtube_visibility = EXCLUDED.youtube_visibility, youtube_publish_at = EXCLUDED.youtube_publish_at,
		    youtube_thumbnail_path = EXCLUDED.youtube_thumbnail_path,
		    youtube_video_url = EXCLUDED.youtube_video_url,
		    error_message = EXCLUDED.error_message,
		    tts_enabled = EXCLUDED.tts_enabled, voice_id = EXCLUDED.voice_id,
		    subtitles_enabled = EXCLUDED.subtitles_enabled, subtitle_style = EXCLUDED.subtitle_style,
		    wait_offsets = EXCLUDED.wait_offsets, rendered_video_seconds = EXCLUDED.rendered_video_seconds,
		    render_quality = EXCLUDED.render_quality,
		    background_music_volume = EXCLUDED.background_music_volume,
		    updated_at = now()`,
		project.ProjectID, project.SagaID, string(project.Status), project.ScriptContent, project.ManimSceneClassName, project.PluginID,
		project.CategoryHint, string(project.ContentLanguage), project.BackgroundMusicPath, scenesJSON, project.RenderedVideoPath, project.VideoPath,
		project.YoutubeTitle, project.YoutubeDescription, tagsJSON, youtubeVisibility, project.YoutubePublishAt,
		project.YoutubeThumbnailPath, project.YoutubeVideoURL, project.ErrorMessage,
		project.TTSEnabled, project.VoiceID, project.SubtitlesEnabled, subtitleStyleJSON,
		waitOffsetsJSON, project.RenderedVideoSeconds, string(project.RenderQuality),
		project.BackgroundMusicVolume)
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
