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
-- CR-025/027/028: the per-project authoring artefacts (Story Architect output,
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
    -- CR-040 FR111: what the orchestrator's projects row used to give the
    -- topic-collision search (CR-028 FR85). The language is sent with the topic.
    language           TEXT NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS project_authoring_topic_lang ON project_authoring (language);

-- CR-027 D9/FR82: one row per LLM call. project_id is nullable and has no FK:
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

-- CR-031: the prompt library. Each pipeline role owns a list of prompts and
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
