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
		SELECT project_id, saga_id, status, script_content, manim_scene_class_name, plugin_id, category_hint, voice_language, video_format_id, video_format_version, review_enabled, beats, validation_warnings,
		       background_music_path, scenes, rendered_video_path, video_path, youtube_title, youtube_description,
		       youtube_tags, youtube_visibility, youtube_publish_at, youtube_thumbnail_path, youtube_channel_id, youtube_video_url, error_message,
		       tts_enabled, voice_id, subtitles_enabled, subtitle_style, wait_offsets, rendered_video_seconds,
		       render_quality, background_music_volume, chapters, caption_path, subtitle_mode, caption_status,
		       intro_enabled, outro_enabled, intro_asset_id, outro_asset_id, layout_marks
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
		chaptersJSON          []byte
		subtitleMode          string
		beatsJSON             []byte
		warningsJSON          []byte
		layoutMarksJSON       []byte
	)
	err := row.Scan(&p.ProjectID, &p.SagaID, &status, &p.ScriptContent, &p.ManimSceneClassName, &p.PluginID, &p.CategoryHint, &voiceLanguage, &p.VideoFormatID, &p.VideoFormatVersion, &p.ReviewEnabled, &beatsJSON, &warningsJSON,
		&p.BackgroundMusicPath, &scenesJSON, &p.RenderedVideoPath, &p.VideoPath, &p.YoutubeTitle, &p.YoutubeDescription,
		&tagsJSON, &youtubeVisibility, &p.YoutubePublishAt, &p.YoutubeThumbnailPath, &p.YoutubeChannelID, &p.YoutubeVideoURL, &p.ErrorMessage,
		&p.TTSEnabled, &p.VoiceID, &p.SubtitlesEnabled, &subtitleStyleJSON, &waitOffsetsJSON, &p.RenderedVideoSeconds,
		&renderQuality, &p.BackgroundMusicVolume, &chaptersJSON, &p.CaptionPath, &subtitleMode, &p.CaptionStatus,
		&p.IntroEnabled, &p.OutroEnabled, &p.IntroAssetID, &p.OutroAssetID, &layoutMarksJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrProjectNotFound
	}
	if err != nil {
		return nil, err
	}

	p.Status = domain.ProjectStatus(status)
	p.RenderQuality = domain.RenderQuality(renderQuality)
	p.ContentLanguage = domain.ContentLanguage(voiceLanguage)
	// '' means this row predates the subtitle_mode column (or was written by
	// code that only knew SubtitlesEnabled) — derive it the one way that
	// boolean ever meant something, rather than leaving Off to silently
	// override a project the Creator actually rendered with burn-in on.
	if subtitleMode == "" {
		p.SubtitleMode = domain.SubtitleModeFromLegacy(p.SubtitlesEnabled)
	} else {
		p.SubtitleMode = domain.SubtitleMode(subtitleMode)
	}
	if youtubeVisibility != nil {
		v := domain.Visibility(*youtubeVisibility)
		p.YoutubeVisibility = &v
	}
	if len(chaptersJSON) > 0 {
		if err := json.Unmarshal(chaptersJSON, &p.Chapters); err != nil {
			return nil, err
		}
	}
	if len(beatsJSON) > 0 {
		if err := json.Unmarshal(beatsJSON, &p.Beats); err != nil {
			return nil, err
		}
	}
	if len(warningsJSON) > 0 {
		if err := json.Unmarshal(warningsJSON, &p.ValidationWarnings); err != nil {
			return nil, err
		}
	}
	if len(layoutMarksJSON) > 0 {
		if err := json.Unmarshal(layoutMarksJSON, &p.LayoutMarks); err != nil {
			return nil, err
		}
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
	// Cả hai chỉ tồn tại để dựng màn duyệt dàn ý (CR-024), nên NULL khi rỗng
	// thay vì một mảng JSON rỗng — dễ đọc hơn khi soi database.
	scenesJSON, err := json.Marshal(project.Scenes)
	if err != nil {
		return err
	}

	// Cả hai chỉ tồn tại để dựng màn duyệt dàn ý (CR-024), nên để NULL khi rỗng
	// thay vì một mảng JSON rỗng — dễ đọc hơn khi soi database.
	var beatsJSON, warningsJSON []byte
	if len(project.Beats) > 0 {
		if beatsJSON, err = json.Marshal(project.Beats); err != nil {
			return err
		}
	}
	if len(project.ValidationWarnings) > 0 {
		if warningsJSON, err = json.Marshal(project.ValidationWarnings); err != nil {
			return err
		}
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

	var chaptersJSON []byte
	if project.Chapters != nil {
		if chaptersJSON, err = json.Marshal(project.Chapters); err != nil {
			return err
		}
	}

	// CR-021: NULL when Rendering sent none (a project rendered before the
	// layout marks existed), which qc_video then reports as not_scored rather
	// than pretending an empty screen.
	var layoutMarksJSON []byte
	if len(project.LayoutMarks) > 0 {
		if layoutMarksJSON, err = json.Marshal(project.LayoutMarks); err != nil {
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
		INSERT INTO projects (project_id, saga_id, status, script_content, manim_scene_class_name, plugin_id, category_hint, voice_language, video_format_id, video_format_version, review_enabled, beats, validation_warnings,
		                       background_music_path, scenes, rendered_video_path, video_path, youtube_title, youtube_description,
		                       youtube_tags, youtube_visibility, youtube_publish_at, youtube_thumbnail_path, youtube_channel_id, youtube_video_url, error_message,
		                       tts_enabled, voice_id, subtitles_enabled, subtitle_style, wait_offsets, rendered_video_seconds,
		                       render_quality, background_music_volume, chapters, caption_path, subtitle_mode, caption_status,
		                       intro_enabled, outro_enabled, intro_asset_id, outro_asset_id, layout_marks, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39,$40,$41,$42,$43, now())
		ON CONFLICT (project_id) DO UPDATE SET
		    saga_id = EXCLUDED.saga_id, status = EXCLUDED.status, script_content = EXCLUDED.script_content,
		    manim_scene_class_name = EXCLUDED.manim_scene_class_name,
		    plugin_id = EXCLUDED.plugin_id, category_hint = EXCLUDED.category_hint, voice_language = EXCLUDED.voice_language,
		    video_format_id = EXCLUDED.video_format_id, video_format_version = EXCLUDED.video_format_version,
		    review_enabled = EXCLUDED.review_enabled, beats = EXCLUDED.beats,
		    validation_warnings = EXCLUDED.validation_warnings,
		    background_music_path = EXCLUDED.background_music_path, scenes = EXCLUDED.scenes,
		    rendered_video_path = EXCLUDED.rendered_video_path,
		    video_path = EXCLUDED.video_path, youtube_title = EXCLUDED.youtube_title,
		    youtube_description = EXCLUDED.youtube_description, youtube_tags = EXCLUDED.youtube_tags,
		    youtube_visibility = EXCLUDED.youtube_visibility, youtube_publish_at = EXCLUDED.youtube_publish_at,
		    youtube_thumbnail_path = EXCLUDED.youtube_thumbnail_path,
		    youtube_channel_id = EXCLUDED.youtube_channel_id,
		    youtube_video_url = EXCLUDED.youtube_video_url,
		    error_message = EXCLUDED.error_message,
		    tts_enabled = EXCLUDED.tts_enabled, voice_id = EXCLUDED.voice_id,
		    subtitles_enabled = EXCLUDED.subtitles_enabled, subtitle_style = EXCLUDED.subtitle_style,
		    wait_offsets = EXCLUDED.wait_offsets, rendered_video_seconds = EXCLUDED.rendered_video_seconds,
		    render_quality = EXCLUDED.render_quality,
		    background_music_volume = EXCLUDED.background_music_volume,
		    chapters = EXCLUDED.chapters,
		    caption_path = EXCLUDED.caption_path,
		    subtitle_mode = EXCLUDED.subtitle_mode,
		    caption_status = EXCLUDED.caption_status,
		    intro_enabled = EXCLUDED.intro_enabled,
		    outro_enabled = EXCLUDED.outro_enabled,
		    intro_asset_id = EXCLUDED.intro_asset_id,
		    outro_asset_id = EXCLUDED.outro_asset_id,
		    layout_marks = EXCLUDED.layout_marks,
		    updated_at = now()`,
		project.ProjectID, project.SagaID, string(project.Status), project.ScriptContent, project.ManimSceneClassName, project.PluginID,
		project.CategoryHint, string(project.ContentLanguage),
		project.VideoFormatID, project.VideoFormatVersion, project.ReviewEnabled, beatsJSON, warningsJSON,
		project.BackgroundMusicPath, scenesJSON, project.RenderedVideoPath, project.VideoPath,
		project.YoutubeTitle, project.YoutubeDescription, tagsJSON, youtubeVisibility, project.YoutubePublishAt,
		project.YoutubeThumbnailPath, project.YoutubeChannelID, project.YoutubeVideoURL, project.ErrorMessage,
		project.TTSEnabled, project.VoiceID, project.SubtitlesEnabled, subtitleStyleJSON,
		waitOffsetsJSON, project.RenderedVideoSeconds, string(project.RenderQuality),
		project.BackgroundMusicVolume, chaptersJSON, project.CaptionPath, string(project.SubtitleMode), project.CaptionStatus,
		project.IntroEnabled, project.OutroEnabled, project.IntroAssetID, project.OutroAssetID, layoutMarksJSON)
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

// RecordVoiceSamples folds one project's measurement into the voice's running
// totals (CR-016 FR43.1).
//
// The upsert adds rather than replaces: a voice's measured rate should settle
// as evidence accumulates, not swing to whatever the last project happened to
// contain.
func (r *ProjectRepository) RecordVoiceSamples(ctx context.Context, voiceID string, words int, seconds float64) error {
	if voiceID == "" || words <= 0 || seconds <= 0 {
		// Nothing measurable — a project with narration disabled, or one whose
		// voice was never recorded. Silently skipping beats storing a row that
		// would drag the average toward zero.
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO voice_calibration (voice_id, sample_count, total_words, total_seconds, updated_at)
		VALUES ($1, 1, $2, $3, now())
		ON CONFLICT (voice_id) DO UPDATE SET
		    sample_count = voice_calibration.sample_count + 1,
		    total_words = voice_calibration.total_words + EXCLUDED.total_words,
		    total_seconds = voice_calibration.total_seconds + EXCLUDED.total_seconds,
		    updated_at = now()
	`, voiceID, words, seconds)
	return err
}

func (r *ProjectRepository) GetVoiceCalibration(ctx context.Context, voiceID string) (domain.VoiceCalibration, error) {
	var c domain.VoiceCalibration
	c.VoiceID = voiceID
	err := r.pool.QueryRow(ctx, `
		SELECT sample_count, total_words, total_seconds FROM voice_calibration WHERE voice_id = $1
	`, voiceID).Scan(&c.SampleCount, &c.TotalWords, &c.TotalSecond)
	if err != nil {
		// An unmeasured voice is the normal starting state, not an error: the
		// caller falls back to the language default.
		return domain.VoiceCalibration{VoiceID: voiceID}, nil
	}
	return c, nil
}

func (r *ProjectRepository) ListVoiceCalibrations(ctx context.Context) ([]domain.VoiceCalibration, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT voice_id, sample_count, total_words, total_seconds FROM voice_calibration ORDER BY voice_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.VoiceCalibration, 0)
	for rows.Next() {
		var c domain.VoiceCalibration
		if err := rows.Scan(&c.VoiceID, &c.SampleCount, &c.TotalWords, &c.TotalSecond); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SeedVideoFormats writes the built-in formats if they are not already there
// (CR-019 FR51.2).
//
// Insert-if-absent rather than upsert: once the Creator has edited a format,
// a restart must not quietly restore the shipped numbers underneath them.
func (r *ProjectRepository) SeedVideoFormats(ctx context.Context) error {
	for _, format := range domain.BuiltinFormats() {
		beats, err := json.Marshal(format.Beats)
		if err != nil {
			return err
		}
		if _, err := r.pool.Exec(ctx, `
			INSERT INTO video_formats (format_id, version, name, min_seconds, max_seconds, beats)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (format_id, version) DO NOTHING
		`, format.ID, format.Version, format.Name, format.MinSeconds, format.MaxSeconds, beats); err != nil {
			return err
		}
	}
	return nil
}

// GetVideoFormat returns one format. version <= 0 means "the newest".
func (r *ProjectRepository) GetVideoFormat(ctx context.Context, formatID string, version int) (domain.VideoFormat, error) {
	if formatID == "" {
		formatID = domain.DefaultVideoFormatID
	}

	query := `
		SELECT format_id, version, name, min_seconds, max_seconds, beats
		FROM video_formats WHERE format_id = $1 AND ($2 <= 0 OR version = $2)
		ORDER BY version DESC LIMIT 1`

	var format domain.VideoFormat
	var beats []byte
	err := r.pool.QueryRow(ctx, query, formatID, version).Scan(
		&format.ID, &format.Version, &format.Name, &format.MinSeconds, &format.MaxSeconds, &beats,
	)
	if err != nil {
		// A project pointing at a format that no longer exists still has to
		// render. Falling back to the built-in default beats failing the saga
		// over a bookkeeping problem the Creator cannot see.
		return domain.FormatVisualFirst7Min, nil
	}
	if err := json.Unmarshal(beats, &format.Beats); err != nil {
		return domain.FormatVisualFirst7Min, nil
	}
	return format, nil
}

// ListVideoFormats returns the newest version of every format.
func (r *ProjectRepository) ListVideoFormats(ctx context.Context) ([]domain.VideoFormat, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ON (format_id) format_id, version, name, min_seconds, max_seconds, beats
		FROM video_formats ORDER BY format_id, version DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.VideoFormat, 0)
	for rows.Next() {
		var format domain.VideoFormat
		var beats []byte
		if err := rows.Scan(&format.ID, &format.Version, &format.Name,
			&format.MinSeconds, &format.MaxSeconds, &beats); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(beats, &format.Beats); err != nil {
			return nil, err
		}
		out = append(out, format)
	}
	return out, rows.Err()
}

// SaveVideoFormat stores a format as a NEW version (CR-019 FR51.6).
//
// Never overwrites: a project rendered against version 3 must keep reporting
// version 3's beats, otherwise editing a format silently rewrites the structure
// of every video already made with it.
func (r *ProjectRepository) SaveVideoFormat(ctx context.Context, format domain.VideoFormat) (domain.VideoFormat, error) {
	beats, err := json.Marshal(format.Beats)
	if err != nil {
		return format, err
	}
	var version int
	err = r.pool.QueryRow(ctx, `
		INSERT INTO video_formats (format_id, version, name, min_seconds, max_seconds, beats)
		VALUES (
		    $1,
		    COALESCE((SELECT MAX(version) FROM video_formats WHERE format_id = $1), 0) + 1,
		    $2, $3, $4, $5
		)
		RETURNING version
	`, format.ID, format.Name, format.MinSeconds, format.MaxSeconds, beats).Scan(&version)
	if err != nil {
		return format, err
	}
	format.Version = version
	return format, nil
}
