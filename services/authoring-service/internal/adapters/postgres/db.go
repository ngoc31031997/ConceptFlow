// Package postgres implements authoring-service's persistence: the prompt
// library, the per-project authoring artefacts and LLM usage. It owns its own
// database (ADR-0013); nothing here reads the orchestrator's `projects` table.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// schema is applied at startup via CREATE TABLE IF NOT EXISTS, like every other
// service in this system.
const schema = `
-- The per-project authoring artefacts (Story Architect output,
-- storyboard, code, topic, working mode, model per step). One row per project.
-- No foreign key: the project lives in the orchestrator's database; deleting a
-- project deletes this row through DELETE /internal/v1/authoring/{id}.
CREATE TABLE IF NOT EXISTS project_authoring (
    project_id         TEXT PRIMARY KEY,
    story_content      TEXT NOT NULL DEFAULT '',
    storyboard_content TEXT NOT NULL DEFAULT '',
    code_content       TEXT NOT NULL DEFAULT '',
    review_content     TEXT NOT NULL DEFAULT '',
    topic              TEXT NOT NULL DEFAULT '',
    authoring_mode     TEXT NOT NULL DEFAULT 'manual',
    story_model        TEXT NOT NULL DEFAULT '',
    storyboard_model   TEXT NOT NULL DEFAULT '',
    code_model         TEXT NOT NULL DEFAULT '',
    -- The topic and its language, for the
    -- topic-collision search. The language is sent with the topic.
    language           TEXT NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS project_authoring_topic_lang ON project_authoring (language);

-- One row per LLM call. project_id is nullable and has no FK:
-- suggest-short-script runs before any project exists, and deleting a project
-- must not erase the record of what it cost.
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
    error_kind        TEXT NOT NULL DEFAULT '',
    phase             TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS llm_usage_created_at_idx ON llm_usage (created_at DESC);
-- A call whose stream was cut (reasoning budget) or that failed comes
-- back with no usage record; its tokens are then unknown, not zero.
ALTER TABLE llm_usage ADD COLUMN IF NOT EXISTS reasoning_chars INTEGER NOT NULL DEFAULT 0;
ALTER TABLE llm_usage ADD COLUMN IF NOT EXISTS usage_reported BOOLEAN NOT NULL DEFAULT true;

-- The prompt library. Each pipeline role owns a list of prompts and
-- exactly one of them is active; is_system rows ship in the binary.
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
CREATE UNIQUE INDEX IF NOT EXISTS prompts_one_active_per_role ON prompts (role) WHERE is_active;
CREATE UNIQUE INDEX IF NOT EXISTS prompts_one_system_per_role ON prompts (role) WHERE is_system;

-- Video archetypes the Story Architect can be told to make. System rows
-- ship in the binary (read-only); the Creator adds their own.
CREATE TABLE IF NOT EXISTS video_archetypes (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    code        TEXT NOT NULL,
    name        TEXT NOT NULL,
    when_to_use TEXT NOT NULL,
    playbook    TEXT NOT NULL,
    is_system   BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS video_archetypes_code_key ON video_archetypes (upper(code));
ALTER TABLE video_archetypes ADD COLUMN IF NOT EXISTS recommended_format_id TEXT NOT NULL DEFAULT '';

-- The illustration library. Folders are shelves; each drawing lives in
-- exactly one. Built-in rows (the illustration kit) carry no code: it ships in the
-- rendering image. The preview is stored for the version it was rendered from,
-- so a stale one is never served after the code changes.
CREATE TABLE IF NOT EXISTS illustration_folders (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    position    INTEGER NOT NULL DEFAULT 1000,
    is_system   BOOLEAN NOT NULL DEFAULT false
);
CREATE TABLE IF NOT EXISTS illustrations (
    id              TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name            TEXT NOT NULL,
    title           TEXT NOT NULL,
    folder_id       TEXT NOT NULL REFERENCES illustration_folders(id),
    tags            TEXT[] NOT NULL DEFAULT '{}',
    description     TEXT NOT NULL DEFAULT '',
    usage           TEXT NOT NULL DEFAULT '',
    code            TEXT NOT NULL DEFAULT '',
    builtin         BOOLEAN NOT NULL DEFAULT false,
    status          TEXT NOT NULL DEFAULT 'draft',
    version         INTEGER NOT NULL DEFAULT 1,
    preview_version INTEGER NOT NULL DEFAULT 0,
    preview_png     BYTEA,
    preview_gif     BYTEA,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS illustrations_name_key ON illustrations (name);
CREATE INDEX IF NOT EXISTS illustrations_folder_idx ON illustrations (folder_id);
-- Style exemplars are read-only rows that carry their code; warnings are the
-- style findings of the current version that did not block saving.
ALTER TABLE illustrations ADD COLUMN IF NOT EXISTS exemplar BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE illustrations ADD COLUMN IF NOT EXISTS warnings JSONB NOT NULL DEFAULT '[]';
-- A Hình mẫu copy points at the drawing it was made from (NULL once
-- that one is deleted); an original exemplar remembers the folder it goes
-- back to when it stops being one.
ALTER TABLE illustrations ADD COLUMN IF NOT EXISTS source_id TEXT REFERENCES illustrations(id) ON DELETE SET NULL;
ALTER TABLE illustrations ADD COLUMN IF NOT EXISTS home_folder_id TEXT;
-- 'figure' (placed in a shot) or 'backdrop' (a whole-frame place in the depth
-- layers of a Scene).
ALTER TABLE illustrations ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'figure';

-- The drawings one video needs, planned from its storyboard. No FK to
-- the project (it lives in the orchestrator); deleting the project deletes
-- these rows with its authoring row.
CREATE TABLE IF NOT EXISTS project_illustrations (
    id              TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    project_id      TEXT NOT NULL,
    position        INTEGER NOT NULL,
    name            TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    folder_id       TEXT NOT NULL DEFAULT '',
    shots           TEXT[] NOT NULL DEFAULT '{}',
    state           TEXT NOT NULL DEFAULT 'planned',
    error           TEXT NOT NULL DEFAULT '',
    illustration_id TEXT REFERENCES illustrations(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS project_illustrations_project_idx ON project_illustrations (project_id, position);
-- 'figure' (drawn for some shots) or 'backdrop' (the place of a whole scene).
ALTER TABLE project_illustrations ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'figure';
-- When the video's drawing list was last planned, so the code step can
-- tell "planned, needs no drawing" from "never planned".
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS illustrations_planned_at TIMESTAMPTZ;
-- sha256 of the storyboard the list was planned from, so a list
-- made from an older storyboard is known to be stale. '' = planned before this
-- existed (or never): no evidence either way, not treated as stale.
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS illustrations_storyboard_sha TEXT NOT NULL DEFAULT '';
-- The code step stored segment by segment, so a
-- failed or interrupted run keeps what it already wrote. No FK, like
-- project_illustrations: DeleteAuthoring removes a project's segments.
CREATE TABLE IF NOT EXISTS authoring_segments (
    project_id    TEXT NOT NULL,
    step          TEXT NOT NULL,
    key           TEXT NOT NULL,
    position      INTEGER NOT NULL,
    kind          TEXT NOT NULL,
    shots         TEXT[] NOT NULL DEFAULT '{}',
    status        TEXT NOT NULL DEFAULT 'pending',
    source        TEXT NOT NULL DEFAULT '',
    fingerprint   TEXT NOT NULL DEFAULT '',
    content       JSONB,
    error_kind    TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    duration_ms   INTEGER NOT NULL DEFAULT 0,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, step, key)
);
CREATE INDEX IF NOT EXISTS authoring_segments_running_idx ON authoring_segments (status) WHERE status = 'running';
-- The shots a failed chunk could not write; its content then holds the
-- shots it did write, so the next run writes only the missing ones.
ALTER TABLE authoring_segments ADD COLUMN IF NOT EXISTS failed_shots TEXT[] NOT NULL DEFAULT '{}';
-- Shots per code segment, chosen by the Creator.
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS code_chunk_shots INTEGER NOT NULL DEFAULT 3;
-- Every failed check of a code run, for statistics. Kept when
-- the project is deleted, like llm_usage.
CREATE TABLE IF NOT EXISTS code_check_diagnostics (
    id          BIGSERIAL PRIMARY KEY,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    project_id  TEXT NOT NULL,
    engine      TEXT NOT NULL,
    phase       TEXT NOT NULL,
    round       INTEGER NOT NULL,
    segment_key TEXT NOT NULL DEFAULT '',
    shot_id     TEXT NOT NULL DEFAULT '',
    kind        TEXT NOT NULL,
    rule        TEXT NOT NULL DEFAULT '',
    message     TEXT NOT NULL,
    line        INTEGER
);
CREATE INDEX IF NOT EXISTS code_check_diagnostics_created_idx ON code_check_diagnostics (created_at DESC);
-- Colours outside the channel palette (S9) are not a warning; drop any such
-- stored warnings so no drawing shows them.
UPDATE illustrations SET warnings = COALESCE((
    SELECT jsonb_agg(w) FROM jsonb_array_elements(warnings) w WHERE w->>'message' NOT LIKE '[S9]%'
), '[]'::jsonb)
WHERE warnings::text LIKE '%[S9]%';
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
