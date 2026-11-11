// Package postgres implements visual-service's persistence. It owns its own
// database (ADR-0049); the project itself lives in the orchestrator and the
// script in authoring-service, so nothing here has a foreign key to them.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// schema is applied at startup via CREATE TABLE IF NOT EXISTS, like every other
// service in this system.
const schema = `
-- The review units of the visual steps, one row per unit, the same shape as
-- authoring-service's. Step 8 uses kind 'visual_concept' (one unit per part of
-- the visual-concept document) and 'visual_review' (key 'main'). input_hashes
-- holds, per input the unit was written from, the hash of that input at the
-- time, so a unit whose input changed reads as stale.
CREATE TABLE IF NOT EXISTS review_units (
    project_id     TEXT NOT NULL,
    kind           TEXT NOT NULL,
    unit_key       TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'pending',
    content        JSONB NOT NULL DEFAULT '{}',
    creator_fields TEXT[] NOT NULL DEFAULT '{}',
    input_hashes   JSONB NOT NULL DEFAULT '{}',
    redo_count     INTEGER NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, kind, unit_key)
);
-- The locked versions of a video's visual concept: an immutable document the
-- later visual steps read, and the script version it was written from.
CREATE TABLE IF NOT EXISTS visual_concept_versions (
    project_id     TEXT NOT NULL,
    version        INTEGER NOT NULL,
    script_version INTEGER NOT NULL,
    document       JSONB NOT NULL,
    note           TEXT NOT NULL DEFAULT '',
    locked_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, version)
);
-- The locked versions of a video's visual plan: an immutable document the later
-- visual steps read, with the script and visual-concept versions it was written
-- from.
CREATE TABLE IF NOT EXISTS visual_plan_versions (
    project_id      TEXT NOT NULL,
    version         INTEGER NOT NULL,
    script_version  INTEGER NOT NULL,
    concept_version INTEGER NOT NULL,
    document        JSONB NOT NULL,
    note            TEXT NOT NULL DEFAULT '',
    locked_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, version)
);
-- The next number of a video's visual-plan unit (VP-001, VP-002...); a number
-- is handed out once and never reused.
CREATE TABLE IF NOT EXISTS visual_plan_seq (
    project_id TEXT PRIMARY KEY,
    next       INTEGER NOT NULL
);
-- Per-video settings of the visual steps. concept_model '' = the default model
-- of the step, kept with the prompt library in llm-service.
CREATE TABLE IF NOT EXISTS project_visual (
    project_id    TEXT PRIMARY KEY,
    concept_model TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- The reference library: text descriptions of a style, shared by every video.
-- Ids are R01, R02... from seq and are never reused.
CREATE SEQUENCE IF NOT EXISTS reference_styles_seq;
CREATE TABLE IF NOT EXISTS reference_styles (
    id          TEXT PRIMARY KEY,
    seq         BIGINT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL,
    tags        TEXT[] NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- name holds style_name, description holds intent; detail holds the other
-- structured fields (source, core_rules, drawing, color_rules, composition,
-- avoid, prompt_template, negative_prompt).
ALTER TABLE reference_styles ADD COLUMN IF NOT EXISTS detail JSONB NOT NULL DEFAULT '{}';
-- Every stored description carries a composition key; a row without one is in
-- an earlier layout that the library no longer reads, and is removed together
-- with the videos' picks of it.
DELETE FROM reference_styles WHERE NOT (detail ? 'composition');
-- Which descriptions a video's visual concept reads.
CREATE TABLE IF NOT EXISTS project_reference_picks (
    project_id TEXT NOT NULL,
    ref_id     TEXT NOT NULL REFERENCES reference_styles (id) ON DELETE CASCADE,
    PRIMARY KEY (project_id, ref_id)
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
