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

-- CR-024: cổng duyệt dàn ý. review_enabled mặc định TRUE — project tạo trước
-- khi cột này tồn tại cũng đi qua cổng, vì đó là hành vi CR muốn.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS review_enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS beats JSONB;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS validation_warnings JSONB;
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

-- CR-023 D7/FR67.1/FR67.2: whether the fixed channel intro/outro is attached
-- at assemble_video. Default TRUE for both — channel identity is opt-out, so
-- a project created before these columns existed also gets it.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS intro_enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS outro_enabled BOOLEAN NOT NULL DEFAULT TRUE;
-- CR-023 D2: the channel_assets id actually resolved and dispatched with this
-- project's assemble_video command, persisted (not re-resolved) so a retry
-- reconstructs the identical payload (Rule 5) rather than looking it up again
-- and potentially disagreeing with what was already sent.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS intro_asset_id TEXT;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS outro_asset_id TEXT;

-- CR-023 correction: orchestrator no longer calls video-assembly over HTTP to
-- find the active intro/outro asset (no such HTTP server exists between
-- backend services). It keeps its own lightweight projection instead, kept
-- current by subscribing to channel_asset_rendered/channel_asset_normalized
-- events, mirroring how handle_step_event.go already folds saga events into
-- projects. No path column here on purpose — video-assembly's own
-- channel_assets table is the only place that resolves asset_id to a real
-- file path.
CREATE TABLE IF NOT EXISTS channel_asset_pointers (
    kind TEXT NOT NULL,
    render_quality TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (kind, render_quality)
);

-- CR-021 FR58/D3: what was on screen at each narration mark, measured by
-- Rendering and carried on rendering_completed. Stored here purely so the
-- qc_video command can be rebuilt from Project alone (Rule 5) rather than
-- needing the original event again — exactly the reason wait_offsets is stored.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS layout_marks JSONB;

-- CR-007 FR19.2/D1/D3: the with-self.clip(...) selections Rendering
-- measured, carried verbatim on rendering_completed exactly like
-- layout_marks — stored so generate_clips can be rebuilt from Project alone
-- (Rule 5). clip_requests holds the Creator-entered selections from POST
-- /v1/projects/{id}/clips separately (D3 merges the two at dispatch time,
-- GUI wins on a matching name). clips is generate_clips's own result, one
-- row's worth of (name, preset, status, output_path, duration_seconds,
-- error_message) entries — a clip-level failure never blocks the saga (D1),
-- so this is just the audit trail the Creator sees on the results screen.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS clip_marks JSONB;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS clip_requests JSONB;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS clips JSONB;

-- CR-007 D5 risk: video-assembly is the only place that knows the channel
-- intro's real length (it resolved intro_asset_id and folded it into
-- effective_lead_in — CR-023), and Orchestrator has no synchronous way to ask
-- it again (CR-023 correction: no HTTP between the two). Stored from
-- video_assembled so generate_clips can shift a Creator's clip selection by
-- the same amount narration/subtitles were already shifted — omit it and a
-- clip is off by exactly the intro's length. 0 when the project has no intro.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS intro_duration_seconds DOUBLE PRECISION NOT NULL DEFAULT 0;

-- CR-007 follow-up: "long" | "short" | "both" — which output(s) this project
-- produces. Default 'long' reproduces the only behaviour that existed before
-- this column did: generate_clips never ran unless a Creator opted in.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS video_output_mode TEXT NOT NULL DEFAULT 'long';

-- CR-026 D1: links two independent projects covering the same topic (a
-- long-form video and a short-form one with its own dedicated script) so
-- the Result screen can show both together. Self-referencing, no FK — the
-- two projects have independent lifecycles.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS companion_project_id TEXT;

-- CR-021 D6/FR61.1: one row per automated QC pass.
--
-- findings is JSONB, not text: FR61.1 asks for machine-readable data and the
-- GUI groups by severity, neither of which a log line supports. History is
-- kept (no primary key on project_id) because the point of the indicate-first
-- mode in D5 is to compare reports across renders while the thresholds are
-- being calibrated.
--
-- overridden_at/overridden_findings are the audit half of FR61.3: a deliberate
-- bypass has to leave a trace, and the trace is only meaningful if it says
-- which findings were waved through — the report's findings can change on the
-- next render, so they are copied, not referenced.
CREATE TABLE IF NOT EXISTS qc_reports (
    project_id TEXT NOT NULL,
    status TEXT NOT NULL,
    reason TEXT,
    findings JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    overridden_at TIMESTAMPTZ NULL,
    overridden_findings JSONB NULL
);

CREATE INDEX IF NOT EXISTS qc_reports_project_created_idx
    ON qc_reports (project_id, created_at DESC);

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

-- CR-025: prompt wording for the 4-role authoring pipeline (Story Architect →
-- Visual Director → Manim Engineer → Script Reviewer), moved out of
-- web-gui's scriptPrompts.ts so an editor can fix wording without a frontend
-- rebuild. version increments on every update (mirrors video_formats'
-- versioning intent, though templates are edited in place rather than
-- appended as new rows — history is not needed here the way it is for
-- rendered projects).
CREATE TABLE IF NOT EXISTS prompt_templates (
    role TEXT NOT NULL,
    language TEXT NOT NULL,
    template_text TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (role, language)
);

-- CR-025 step 1 (Story Architect): the pasted story outline a Creator gets
-- back from the external AI, saved server-side so the wizard can hand it to
-- the next pipeline step (Visual Director) via {{previous_output}}. A
-- separate table rather than a projects column: Project's Save() is one large
-- positional INSERT/UPDATE (49 columns) shared by every saga step, and this
-- field is authoring-time-only data with a completely different write path
-- (one Creator action, not saga event folding) — bolting it onto that query
-- would risk misaligning every existing positional parameter.
CREATE TABLE IF NOT EXISTS project_authoring (
    project_id TEXT PRIMARY KEY,
    story_content TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- CR-025 step 2 (Visual Director): the pasted storyboard a Creator gets back
-- from the external AI, same reasoning and same table as story_content above
-- (one row per project, authoring-time-only data) — a second column rather
-- than a second table since it shares the exact same key and lifecycle as
-- story_content.
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS storyboard_content TEXT NOT NULL DEFAULT '';

-- CR-025 step 3 (Manim Engineer): the pasted Manim code a Creator gets back
-- from the external AI, same reasoning/table as story_content/
-- storyboard_content above.
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS code_content TEXT NOT NULL DEFAULT '';

-- CR-025 step 4 (Script Reviewer): the pasted PASS/REVISE verdict text a
-- Creator gets back from the external AI, same reasoning/table as the columns
-- above.
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS review_content TEXT NOT NULL DEFAULT '';
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
