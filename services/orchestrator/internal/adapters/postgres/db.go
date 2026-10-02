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

-- Which connected channel a project publishes to. Same reason as
-- above — an already-bootstrapped database never re-runs CREATE TABLE.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS youtube_channel_id TEXT;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS youtube_thumbnail_path TEXT;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS manim_scene_class_name TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN IF NOT EXISTS rendered_video_path TEXT;
-- Narration and subtitles are switchable per project. tts_enabled
-- defaults to true so projects created before this column existed keep the
-- narrated behaviour they were rendered with.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS tts_enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS voice_id TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN IF NOT EXISTS subtitles_enabled BOOLEAN NOT NULL DEFAULT FALSE;

-- What each voice was actually measured reading at.
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

-- Hình dạng lặp lại của một video, dưới dạng dữ liệu sửa được.
--
-- Là bảng chứ không phải hằng số trong mã nguồn vì beat nào một chủ đề cần thì
-- thay đổi rất nhiều — một bộ beat hardcode sẽ sai ngay ở chủ đề đầu tiên không
-- vừa khuôn. Cột version để một project render tháng trước vẫn
-- báo đúng cấu trúc nó thực sự được dựng theo.
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

-- Cổng duyệt dàn ý. review_enabled mặc định TRUE — project tạo trước
-- khi cột này tồn tại cũng đi qua cổng, vì đó là hành vi CR muốn.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS review_enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS beats JSONB;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS validation_warnings JSONB;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS subtitle_style JSONB;
-- Where each narration segment actually begins in the rendered video,
-- as measured by Rendering. Projects rendered before this column existed have
-- NULL here; assemble_video then falls back to offset 0 for every segment,
-- which reproduces the old (desynchronised) behaviour, so such a project needs
-- re-rendering rather than re-assembling.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS wait_offsets JSONB;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS rendered_video_seconds DOUBLE PRECISION NOT NULL DEFAULT 0;
-- resolution/framerate for this project's render. Defaults to 1080p60
-- so projects created before this column existed are upgraded rather than
-- pinned to the old hardcoded 720p30.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS render_quality TEXT NOT NULL DEFAULT '1080p60';
-- Creator-chosen background music level. 0 means unset, which assembly
-- reads as the 0.2 the level was fixed at before this was adjustable.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS background_music_volume DOUBLE PRECISION NOT NULL DEFAULT 0;
-- Font for text drawn inside a Remotion video. '' means DefaultVideoFont.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS video_font TEXT NOT NULL DEFAULT '';
-- Chapter markers from the script. Timestamps are not stored — they are
-- derived from wait_offsets, so a re-render moves the chapters with the video.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS chapters JSONB;
-- The .srt caption track Video Assembly wrote alongside video_path,
-- when subtitle_mode asked for one. NULL for every project rendered before
-- this column existed, and for one where subtitles were off or burn-in only —
-- Publisher already treats a NULL/absent caption_path as "nothing to upload"
-- the same way it does youtube_thumbnail_path.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS caption_path TEXT;
-- Which of the four delivery modes this project renders
-- subtitles with. Empty string for every row predating this column —
-- project_repository.go's Get() derives it from subtitles_enabled in that
-- case (domain.SubtitleModeFromLegacy), never left blank downstream.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS subtitle_mode TEXT NOT NULL DEFAULT '';
-- Mirrors the Publisher's PublishResult.caption_status, so a
-- silently skipped or failed caption upload is visible on the project.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS caption_status TEXT;
-- Append-only trace of failures (authoring runs today), newest 100 kept. A log
-- for humans: nothing reads it to make a decision. See application.ProjectError.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS project_errors JSONB NOT NULL DEFAULT '[]';

-- Whether the fixed channel intro/outro is attached
-- at assemble_video. Default TRUE for both — channel identity is opt-out, so
-- a project created before these columns existed also gets it.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS intro_enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS outro_enabled BOOLEAN NOT NULL DEFAULT TRUE;
-- The channel_assets id actually resolved and dispatched with this
-- project's assemble_video command, persisted (not re-resolved) so a retry
-- reconstructs the identical payload (Rule 5) rather than looking it up again
-- and potentially disagreeing with what was already sent.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS intro_asset_id TEXT;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS outro_asset_id TEXT;

-- Orchestrator does not call video-assembly over HTTP to
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

-- What was on screen at each narration mark, measured by
-- Rendering and carried on rendering_completed. Stored here purely so the
-- qc_video command can be rebuilt from Project alone (Rule 5) rather than
-- needing the original event again — exactly the reason wait_offsets is stored.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS layout_marks JSONB;

-- "long" | "short" — the 16:9 long-form video or the vertical short this
-- project produces. Default 'long' is the mode of a project that never chose.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS video_output_mode TEXT NOT NULL DEFAULT 'long';

-- Links two independent projects covering the same topic (a
-- long-form video and a short-form one with its own dedicated script) so
-- the Result screen can show both together. Self-referencing, no FK — the
-- two projects have independent lifecycles.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS companion_project_id TEXT;

-- Remotion engine: which rendering backend (Manim or Remotion) runs this
-- project's script. Default 'manim' reproduces the only behaviour that
-- existed before this column did.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS render_engine TEXT NOT NULL DEFAULT 'manim';

-- Wizard progress: the furthest step (1-3) the Creator confirmed with "Tiếp
-- tục". Steps 4-7 are derived from status, so they are never stored here.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS wizard_step INTEGER NOT NULL DEFAULT 1;

-- Wizard position: the last wizard screen (route) the Creator had open while the
-- project was a draft, so "Chi tiết" reopens exactly there rather than at the
-- furthest screen that happens to have content. '' = never recorded.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS wizard_route TEXT NOT NULL DEFAULT '';

-- One row per automated QC pass.
--
-- findings is JSONB, not text: the report must be machine-readable data and the
-- GUI groups by severity, neither of which a log line supports. History is
-- kept (no primary key on project_id) because the point of the indicate-first
-- mode is to compare reports across renders while the thresholds are
-- being calibrated.
--
-- overridden_at/overridden_findings are the audit trail of a QC override: a deliberate
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

-- Step 1 (Story Architect): the pasted story outline a Creator gets
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

-- Step 2 (Visual Director): the pasted storyboard a Creator gets back
-- from the external AI, same reasoning and same table as story_content above
-- (one row per project, authoring-time-only data) — a second column rather
-- than a second table since it shares the exact same key and lifecycle as
-- story_content.
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS storyboard_content TEXT NOT NULL DEFAULT '';

-- Step 3 (Manim Engineer): the pasted Manim code a Creator gets back
-- from the external AI, same reasoning/table as story_content/
-- storyboard_content above.
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS code_content TEXT NOT NULL DEFAULT '';

-- Step 4 (Script Reviewer), không còn dùng: cột giữ lại cho dữ liệu cũ,
-- không gì đọc/ghi. The pasted PASS/REVISE verdict text a
-- Creator gets back from the external AI, same reasoning/table as the columns
-- above.
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS review_content TEXT NOT NULL DEFAULT '';

-- The project's topic. Until now the topic lived only in the
-- browser (ProjectDraftContext + localStorage) and was interpolated into
-- {{topic}} by scriptPrompts.ts on the client, so the server had no way to
-- render a prompt at all — which the server-side authoring chain needs.
--
-- Same table as the four *_content columns above and for the same reason:
-- authoring-time-only data, one row per project, written by a Creator action
-- rather than by folding a saga event. Deliberately NOT a column on projects,
-- whose single long positional UPDATE would put every existing parameter at
-- risk of misalignment for the sake of one authoring field.
--
-- Projects without a saved topic keep '' here; their rendered prompt then
-- carries the same "paste your topic here" placeholder the GUI shows today.
-- No attempt is made to guess a topic out of story_content.
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS topic TEXT NOT NULL DEFAULT '';

-- How the Creator works step 1 — 'manual' (copy each prompt out
-- to ChatGPT/Claude/Gemini and paste the answer back) or 'ai' (the server
-- renders the prompt and calls the provider itself).
--
-- Server-side, not just in the browser's draft: the choice governs all four
-- tabs of step 1 and a project can be picked up again on any of them, from
-- another browser or after this stack restarts. localStorage answers none of
-- those — it is per-browser and gone the moment someone clears it, and a
-- Creator who chose 'ai' on 1a would silently be back to copy-and-paste on 1c.
--
-- Same table and same reasoning as the columns above: one row per project,
-- authoring-time-only, written by a Creator action rather than by folding a
-- saga event. Every project that existed before this column gets 'manual',
-- which is exactly what it was doing.
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS authoring_mode TEXT NOT NULL DEFAULT 'manual';

-- Model per step: which Hive model each of the three
-- authoring tabs (1a story / 1b storyboard / 1c code) calls, instead of the
-- one model HIVE_MODEL hardcodes for the whole deployment. "" means "server
-- default" — same meaning, same reasoning as authoring_mode's own default
-- above, and the value every project had before this picker existed.
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS story_model TEXT NOT NULL DEFAULT '';
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS storyboard_model TEXT NOT NULL DEFAULT '';
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS code_model TEXT NOT NULL DEFAULT '';

-- Projects never had a created_at column — every existing
-- consumer of this table either already knew its own creation time (the
-- Creator, from the wizard) or didn't need it. The topic collision list does
-- ("tạo lúc ..."), so it gets one now. DEFAULT now() means every row that
-- already existed when this migration runs gets the migration's timestamp,
-- not its true creation time — acceptable here: this column is a display/
-- sort convenience for the warning banner, not data anything else keys off.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- project_authoring was deliberately FK-less because a
-- project could outlive its own existence check — authoring rows were
-- written under a project_id the projects table might never see (the
-- review that found this: docs/review/data-flow-review.md, "Rủi ro"). Now
-- that POST /v1/projects always creates the projects row FIRST,
-- the FK is safe to add — NOT VALID so it only checks rows written from now
-- on and does not fail startup over authoring rows orphaned before this
-- migration ran (those are cleaned up by hand, no automatic sweep).
-- VALIDATE CONSTRAINT can be run once the backlog is
-- confirmed clean.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'project_authoring_project_id_fkey'
    ) THEN
        ALTER TABLE project_authoring
            ADD CONSTRAINT project_authoring_project_id_fkey
            FOREIGN KEY (project_id) REFERENCES projects (project_id) ON DELETE CASCADE
            NOT VALID;
    END IF;
END $$;

-- The append-only project_authoring_history table was never
-- read by anything; authoring saves now just overwrite, and clear the steps
-- built on the one that changed.
DROP TABLE IF EXISTS project_authoring_history;

-- One row per LLM call, so the Creator can see spend in the
-- web GUI instead of on a provider dashboard. This is the first paid service
-- in the pipeline: measure first, enforce later.
-- No spending cap here on purpose — a cap set before anyone knows the real
-- numbers is how a gate loses its credibility.
--
-- Prompts and answers are NOT stored: project_authoring already holds them,
-- and a second copy would double the data at risk for no new insight.
--
-- reasoning_tokens is its own column rather than folded into completion:
-- measured, glm-5.3-flash spent 66 of 122 completion tokens reasoning before
-- answering a one-sentence question. A screen that hides that cannot
-- explain why one model costs twice another for the same visible output.
--
-- project_id is nullable and carries NO foreign key: deleting a project must
-- not erase the record of what it cost.
CREATE TABLE IF NOT EXISTS llm_usage (
    id                BIGSERIAL PRIMARY KEY,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    provider          TEXT NOT NULL,
    model             TEXT NOT NULL,
    role              TEXT NOT NULL DEFAULT '',
    step              TEXT NOT NULL DEFAULT '',
    project_id        TEXT,
    prompt_tokens     INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    reasoning_tokens  INTEGER NOT NULL DEFAULT 0,
    cached_tokens     INTEGER NOT NULL DEFAULT 0,
    duration_ms       INTEGER NOT NULL DEFAULT 0,
    ok                BOOLEAN NOT NULL,
    error_kind        TEXT NOT NULL DEFAULT ''
);

-- Every read of this table is "recent first" or "the last N days", so the
-- index matches the only access pattern there is.
CREATE INDEX IF NOT EXISTS llm_usage_created_at_idx ON llm_usage (created_at DESC);

-- The code step is now several calls (layout/cast, one per chunk,
-- repairs). phase says which, so a step's cost can be broken down; '' for every
-- call that is one call for its step.
ALTER TABLE llm_usage ADD COLUMN IF NOT EXISTS phase TEXT NOT NULL DEFAULT '';

-- The prompt library. Each pipeline role owns a list of prompts and
-- exactly one of them is active. A row with is_system ships in the binary
-- (seeded on every start, read-only); the rest belong to the Creator. This
-- replaces prompt_templates + prompt_overrides, which PurgeLegacyPrompts drops.
CREATE TABLE IF NOT EXISTS prompts (
    id            TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    role          TEXT NOT NULL,
    name          TEXT NOT NULL,
    template_text TEXT NOT NULL,
    is_system     BOOLEAN NOT NULL DEFAULT false,
    is_active     BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- The database itself refuses a second active row, or a second shipped row,
-- for one role — application code only has to switch them in the right order.
CREATE UNIQUE INDEX IF NOT EXISTS prompts_one_active_per_role ON prompts (role) WHERE is_active;
CREATE UNIQUE INDEX IF NOT EXISTS prompts_one_system_per_role ON prompts (role) WHERE is_system;

-- Append-only journey of every project through the 13-step flow (a status
-- change, or an authoring run finishing). Read by the "Nhật ký" screen; no
-- code path makes a decision from it. duration_ms on a status change is the
-- time the project spent in from_status.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS forked_from TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS project_events (
    id                BIGSERIAL PRIMARY KEY,
    project_id        TEXT NOT NULL,
    at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    flow_step         INT NOT NULL,
    run_state         TEXT NOT NULL,
    source            TEXT NOT NULL,
    from_status       TEXT NOT NULL DEFAULT '',
    to_status         TEXT NOT NULL DEFAULT '',
    duration_ms       BIGINT NOT NULL DEFAULT 0,
    detail            TEXT NOT NULL DEFAULT '',
    content_chars     INT NOT NULL DEFAULT 0,
    prompt_tokens     INT NOT NULL DEFAULT 0,
    completion_tokens INT NOT NULL DEFAULT 0
);
ALTER TABLE project_events ADD COLUMN IF NOT EXISTS from_flow_step INT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS project_events_project_at ON project_events (project_id, at);
CREATE INDEX IF NOT EXISTS project_events_at ON project_events (at DESC);

-- One-off data migrations that CREATE/ALTER above cannot express (they only
-- add structure, never reshape existing rows). Each is guarded by an id in
-- this table so it runs exactly once no matter how many times the schema
-- string above is re-executed at startup.
CREATE TABLE IF NOT EXISTS schema_migrations (
    id         TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Illustrations has its own numbered flow step (6). This migration shifts
-- the stored project_events rows of every step from Validate through Publish
-- up by one to match. A DO block is one statement, hence one transaction,
-- and the schema_migrations guard keeps a second run from shifting twice.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM schema_migrations WHERE id = 'cr046_illustrations_flow_step') THEN
        UPDATE project_events SET flow_step = flow_step + 1 WHERE flow_step >= 6;
        UPDATE project_events SET from_flow_step = from_flow_step + 1 WHERE from_flow_step >= 6;
        INSERT INTO schema_migrations (id) VALUES ('cr046_illustrations_flow_step');
    END IF;
END $$;

-- Illustrations runs BEFORE Code (Code reads its drawings),
-- so the two numbers swap: Illustrations=5, Code=6. A CASE swaps both in one
-- pass, so 5->6 and 6->5 never collide.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM schema_migrations WHERE id = 'cr046b_swap_code_illustrations') THEN
        UPDATE project_events SET flow_step = CASE flow_step WHEN 5 THEN 6 WHEN 6 THEN 5 ELSE flow_step END
            WHERE flow_step IN (5, 6);
        UPDATE project_events SET from_flow_step = CASE from_flow_step WHEN 5 THEN 6 WHEN 6 THEN 5 ELSE from_flow_step END
            WHERE from_flow_step IN (5, 6);
        INSERT INTO schema_migrations (id) VALUES ('cr046b_swap_code_illustrations');
    END IF;
END $$;

-- The flow has no vertical-clip step any more: Result is 12 and Publish 13.
-- Stored rows of 13/14 shift down by one; a row at the old step 12 belongs
-- to the step before it, Merge (11). A CASE moves all three in one pass.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM schema_migrations WHERE id = 'remove_clip_step_flow_numbers') THEN
        UPDATE project_events SET flow_step = CASE flow_step WHEN 12 THEN 11 WHEN 13 THEN 12 WHEN 14 THEN 13 END
            WHERE flow_step IN (12, 13, 14);
        UPDATE project_events SET from_flow_step = CASE from_flow_step WHEN 12 THEN 11 WHEN 13 THEN 12 WHEN 14 THEN 13 END
            WHERE from_flow_step IN (12, 13, 14);
        INSERT INTO schema_migrations (id) VALUES ('remove_clip_step_flow_numbers');
    END IF;
END $$;

-- 'both' (the long video plus vertical clips cut from it) is no longer an
-- output mode; a row still holding it is read as the long video it always
-- produced.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM schema_migrations WHERE id = 'remove_output_mode_both') THEN
        UPDATE projects SET video_output_mode = 'long' WHERE video_output_mode = 'both';
        INSERT INTO schema_migrations (id) VALUES ('remove_output_mode_both');
    END IF;
END $$;

-- Columns of the removed vertical-clip step.
ALTER TABLE projects
    DROP COLUMN IF EXISTS clip_marks,
    DROP COLUMN IF EXISTS clip_requests,
    DROP COLUMN IF EXISTS clips,
    DROP COLUMN IF EXISTS intro_duration_seconds;
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
