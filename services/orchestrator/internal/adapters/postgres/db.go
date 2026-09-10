// Package postgres implements the persistence adapter: pgx-backed
// ProjectRepositoryPort, the Inbox (dedupe incoming events) and the Outbox
// (guarantee outgoing command delivery — ADR-0019: unlike every other unit
// in this system, this Outbox holds COMMANDS, not events).
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// schema is applied at startup via CREATE TABLE IF NOT EXISTS, consistent
// with how Units 2-7 self-bootstrap their schema (no separate migration
// tool for this system — Step 16 of the code generation plan).
const schema = `
CREATE TABLE IF NOT EXISTS projects (
    project_id TEXT PRIMARY KEY,
    saga_id TEXT NOT NULL,
    status TEXT NOT NULL,
    script_content TEXT NOT NULL DEFAULT '',
    manim_scene_class_name TEXT NOT NULL DEFAULT '',
    plugin_id TEXT NOT NULL DEFAULT '',
    category_hint TEXT NOT NULL DEFAULT '',
    voice_language TEXT NOT NULL DEFAULT '',
    background_music_path TEXT,
    scenes JSONB NOT NULL DEFAULT '[]',
    rendered_video_path TEXT,
    video_path TEXT,
    youtube_title TEXT,
    youtube_description TEXT,
    youtube_tags JSONB,
    youtube_visibility TEXT,
    youtube_publish_at TEXT,
    youtube_thumbnail_path TEXT,
    youtube_channel_id TEXT,
    youtube_video_url TEXT,
    caption_path TEXT,
    error_message TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Added after the initial CREATE TABLE shipped without it — CREATE TABLE IF
-- NOT EXISTS above is a no-op against an already-bootstrapped database, so
-- existing deployments need this explicit ALTER to pick up the column.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS youtube_publish_at TEXT;

-- CR-012: which connected channel a project publishes to. Same reason as
-- above — an already-bootstrapped database never re-runs CREATE TABLE.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS youtube_channel_id TEXT;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS youtube_thumbnail_path TEXT;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS manim_scene_class_name TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN IF NOT EXISTS rendered_video_path TEXT;
-- CR-001: narration and subtitles are switchable per project. tts_enabled
-- defaults to true so projects created before this column existed keep the
-- narrated behaviour they were rendered with.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS tts_enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS voice_id TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN IF NOT EXISTS subtitles_enabled BOOLEAN NOT NULL DEFAULT FALSE;

-- CR-016 FR43: what each voice was actually measured reading at.
--
-- The words-per-minute constants in the domain are a guess that had never been
-- checked. Every synthesis run is a free chance to check it: the Orchestrator
-- knows both the text it sent and the real audio duration that came back.
-- Kept per voice, not per language — two Vietnamese voices read at visibly
-- different speeds, and averaging them cancels out what is being measured.
CREATE TABLE IF NOT EXISTS voice_calibration (
    voice_id TEXT PRIMARY KEY,
    sample_count INTEGER NOT NULL DEFAULT 0,
    total_words BIGINT NOT NULL DEFAULT 0,
    total_seconds DOUBLE PRECISION NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- CR-019: hình dạng lặp lại của một video, dưới dạng dữ liệu sửa được.
--
-- Là bảng chứ không phải hằng số trong mã nguồn vì beat nào một chủ đề cần thì
-- thay đổi rất nhiều — một bộ beat hardcode sẽ sai ngay ở chủ đề đầu tiên không
-- vừa khuôn (FR51.4/FR51.5). Cột version để một project render tháng trước vẫn
-- báo đúng cấu trúc nó thực sự được dựng theo (FR51.6).
CREATE TABLE IF NOT EXISTS video_formats (
    format_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    name TEXT NOT NULL,
    min_seconds DOUBLE PRECISION NOT NULL,
    max_seconds DOUBLE PRECISION NOT NULL,
    beats JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (format_id, version)
);

-- Format Creator chọn cho project, và phiên bản format tại thời điểm chạy.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS video_format_id TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN IF NOT EXISTS video_format_version INTEGER NOT NULL DEFAULT 0;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS subtitle_style JSONB;
-- CR-002: where each narration segment actually begins in the rendered video,
-- as measured by Rendering. Projects rendered before this column existed have
-- NULL here; assemble_video then falls back to offset 0 for every segment,
-- which reproduces the old (desynchronised) behaviour, so such a project needs
-- re-rendering rather than re-assembling.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS wait_offsets JSONB;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS rendered_video_seconds DOUBLE PRECISION NOT NULL DEFAULT 0;
-- CR-004: resolution/framerate for this project's render. Defaults to 1080p60
-- so projects created before this column existed are upgraded rather than
-- pinned to the old hardcoded 720p30.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS render_quality TEXT NOT NULL DEFAULT '1080p60';
-- CR-005: Creator-chosen background music level. 0 means unset, which assembly
-- reads as the 0.2 the level was fixed at before this was adjustable.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS background_music_volume DOUBLE PRECISION NOT NULL DEFAULT 0;
-- CR-006: chapter markers from the script. Timestamps are not stored — they are
-- derived from wait_offsets, so a re-render moves the chapters with the video.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS chapters JSONB;
-- CR-015: the .srt caption track Video Assembly wrote alongside video_path,
-- when subtitle_mode asked for one. NULL for every project rendered before
-- this column existed, and for one where subtitles were off or burn-in only —
-- Publisher already treats a NULL/absent caption_path as "nothing to upload"
-- the same way it does youtube_thumbnail_path.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS caption_path TEXT;
-- CR-015 FR41: which of the four delivery modes this project renders
-- subtitles with. Empty string for every row predating this column —
-- project_repository.go's Get() derives it from subtitles_enabled in that
-- case (domain.SubtitleModeFromLegacy), never left blank downstream.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS subtitle_mode TEXT NOT NULL DEFAULT '';
-- CR-015 FR39.4: mirrors the Publisher's PublishResult.caption_status, so a
-- silently skipped or failed caption upload is visible on the project.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS caption_status TEXT;

CREATE TABLE IF NOT EXISTS saga_steps (
    saga_id TEXT NOT NULL,
    step_name TEXT NOT NULL,
    status TEXT NOT NULL,
    error_message TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (saga_id, step_name)
);

-- outbox_events: queues COMMANDS to send (not events, unlike Units 2-7's
-- Outbox) — see ADR-0019 for why the table name is nonetheless kept
-- consistent with the rest of the system.
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY,
    routing_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ NULL
);

-- processed_messages: Inbox, dedupes incoming EVENT message_id.
CREATE TABLE IF NOT EXISTS processed_messages (
    message_id UUID PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

// NewPool opens a pgx connection pool against databaseURL with the given max
// connections, then bootstraps the schema.
func NewPool(ctx context.Context, databaseURL string, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = maxConns

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}

	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, fmt.Errorf("bootstrap schema: %w", err)
	}
	return pool, nil
}
