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
    youtube_video_url TEXT,
    error_message TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Added after the initial CREATE TABLE shipped without it — CREATE TABLE IF
-- NOT EXISTS above is a no-op against an already-bootstrapped database, so
-- existing deployments need this explicit ALTER to pick up the column.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS youtube_publish_at TEXT;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS youtube_thumbnail_path TEXT;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS manim_scene_class_name TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN IF NOT EXISTS rendered_video_path TEXT;
-- CR-001: narration and subtitles are switchable per project. tts_enabled
-- defaults to true so projects created before this column existed keep the
-- narrated behaviour they were rendered with.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS tts_enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS voice_id TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN IF NOT EXISTS subtitles_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS subtitle_style JSONB;
-- CR-002: where each narration segment actually begins in the rendered video,
-- as measured by Rendering. Projects rendered before this column existed have
-- NULL here; assemble_video then falls back to offset 0 for every segment,
-- which reproduces the old (desynchronised) behaviour, so such a project needs
-- re-rendering rather than re-assembling.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS wait_offsets JSONB;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS rendered_video_seconds DOUBLE PRECISION NOT NULL DEFAULT 0;

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
