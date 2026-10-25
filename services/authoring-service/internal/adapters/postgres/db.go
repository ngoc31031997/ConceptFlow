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

-- Model calls are logged by llm-service in its own database.
DROP TABLE IF EXISTS llm_usage;

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
-- the project is deleted.
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
-- One reviewable piece of a project's content (a brief, later facts, beats,
-- lines...), addressed as kind:unit_key. creator_fields are the parts of
-- content the Creator wrote, which an AI redo keeps verbatim; redo_count
-- counts AI redos, for the first-time-approval rate. No FK, like
-- authoring_segments: DeleteAuthoring removes a project's units. See ADR-0035.
CREATE TABLE IF NOT EXISTS review_units (
    project_id     TEXT NOT NULL,
    kind           TEXT NOT NULL,
    unit_key       TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'pending',
    content        JSONB NOT NULL DEFAULT '{}',
    creator_fields TEXT[] NOT NULL DEFAULT '{}',
    redo_count     INTEGER NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, kind, unit_key)
);
-- Model that suggests the publish title and description ('' = the default).
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS publish_model TEXT NOT NULL DEFAULT '';
-- Settings of the whole app, one row each (key, value).
CREATE TABLE IF NOT EXISTS app_settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
-- Model the idea step writes the creative brief with ('' = server default).
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS brief_model TEXT NOT NULL DEFAULT '';
-- The video's target length chosen at the idea step ('3-4', '4-5', '5-7',
-- '7-10' minutes; '' for a short, which is always under a minute).
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS target_duration TEXT NOT NULL DEFAULT '';
-- md5 of the approved brief the saved story outline was written from, so an
-- outline written from an older brief (or before any brief) is known to be
-- stale. '' = no approved brief at the time.
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS story_brief_sha TEXT NOT NULL DEFAULT '';
-- The visual languages of the channel: the kit that paints the forms
-- (kit_key) and the rule texts, versioned. Exactly one language is active; a
-- project pins one (id, version) when its authoring row is created.
CREATE TABLE IF NOT EXISTS visual_languages (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    note        TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    kit_key     TEXT NOT NULL,
    is_system   BOOLEAN NOT NULL DEFAULT false,
    is_active   BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS visual_languages_one_active ON visual_languages ((true)) WHERE is_active;
CREATE UNIQUE INDEX IF NOT EXISTS visual_languages_name_key ON visual_languages (lower(name));
-- Append-only: editing a language's rules adds a version, never rewrites one.
CREATE TABLE IF NOT EXISTS visual_language_versions (
    language_id        TEXT NOT NULL REFERENCES visual_languages(id) ON DELETE CASCADE,
    version            INTEGER NOT NULL,
    illustration_rules TEXT NOT NULL,
    backdrop_rules     TEXT NOT NULL,
    scene_rules        TEXT NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (language_id, version)
);
-- Sample scenes of a kit, rendered by the rendering service into its shared
-- stills directory; kit_version says which version of the kit drew them.
CREATE TABLE IF NOT EXISTS visual_language_samples (
    kit_key     TEXT NOT NULL,
    scene       TEXT NOT NULL,
    frame       TEXT NOT NULL,
    kit_version INTEGER NOT NULL,
    file        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (kit_key, scene, frame)
);
-- The language version a project is pinned to (NULL = not pinned: its drawing,
-- check and render steps refuse to run).
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS visual_language_id TEXT REFERENCES visual_languages(id);
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS visual_language_version INTEGER;
-- Pins the active language the moment a project's authoring row is created, on
-- whichever path creates it.
CREATE OR REPLACE FUNCTION pin_visual_language() RETURNS trigger AS $$
BEGIN
    IF NEW.visual_language_id IS NULL THEN
        SELECT l.id, (SELECT max(version) FROM visual_language_versions WHERE language_id = l.id)
          INTO NEW.visual_language_id, NEW.visual_language_version
          FROM visual_languages l WHERE l.is_active;
    END IF;
    RETURN NEW;
END
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS project_authoring_pin_language ON project_authoring;
CREATE TRIGGER project_authoring_pin_language BEFORE INSERT ON project_authoring
    FOR EACH ROW EXECUTE FUNCTION pin_visual_language();
-- The kit a library drawing was drawn for, and how much detail it has
-- (rich / medium / silhouette: the tier it was first drawn as).
ALTER TABLE illustrations ADD COLUMN IF NOT EXISTS kit_key TEXT NOT NULL DEFAULT '';
ALTER TABLE illustrations ADD COLUMN IF NOT EXISTS detail_level TEXT NOT NULL DEFAULT 'medium';
-- The role of a drawing in this video (hero / support / background) and the
-- AI's one-line reason for it.
ALTER TABLE project_illustrations ADD COLUMN IF NOT EXISTS tier TEXT NOT NULL DEFAULT 'support';
ALTER TABLE project_illustrations ADD COLUMN IF NOT EXISTS tier_reason TEXT NOT NULL DEFAULT '';
-- The unit a review unit was built from (kind:key) and the hash of that
-- unit's content at the time; a unit whose source has changed since is stale.
ALTER TABLE review_units ADD COLUMN IF NOT EXISTS input_ref TEXT NOT NULL DEFAULT '';
ALTER TABLE review_units ADD COLUMN IF NOT EXISTS input_sha TEXT NOT NULL DEFAULT '';
-- Model the research step writes its questions with ('' = server default).
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS research_model TEXT NOT NULL DEFAULT '';
-- Model the story step proposes approaches and writes the frame with ('' = server default).
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS story_frame_model TEXT NOT NULL DEFAULT '';
-- Model the beat sheet step writes the beats with ('' = server default).
ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS beat_sheet_model TEXT NOT NULL DEFAULT '';
-- The reading of each script line by a voice: which wording and voice, and the
-- file the voice service wrote (a path under /shared, never the audio).
CREATE TABLE IF NOT EXISTS voice_lines (
    project_id   TEXT NOT NULL,
    line_id      TEXT NOT NULL,
    text         TEXT NOT NULL,
    text_sha     TEXT NOT NULL,
    voice_id     TEXT NOT NULL DEFAULT '',
    language     TEXT NOT NULL DEFAULT 'vi',
    status       TEXT NOT NULL DEFAULT 'pending',
    audio_path   TEXT NOT NULL DEFAULT '',
    duration_s   DOUBLE PRECISION NOT NULL DEFAULT 0,
    cache_key    TEXT NOT NULL DEFAULT '',
    cache_hit    BOOLEAN NOT NULL DEFAULT false,
    error        TEXT NOT NULL DEFAULT '',
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, line_id)
);
-- A locked version of the script: an immutable snapshot of the parts the steps
-- after the script read, each with its hash.
CREATE TABLE IF NOT EXISTS script_versions (
    project_id TEXT NOT NULL,
    version    INTEGER NOT NULL,
    note       TEXT NOT NULL DEFAULT '',
    locked_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    parts      JSONB NOT NULL,
    PRIMARY KEY (project_id, version)
);
-- One row per answer the Creator pasted into the research step, numbered per
-- project. Holds what was read from it, never the pasted text itself.
CREATE TABLE IF NOT EXISTS research_pastes (
    project_id TEXT NOT NULL,
    n          INTEGER NOT NULL,
    at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    added      TEXT[] NOT NULL DEFAULT '{}',
    updated    TEXT[] NOT NULL DEFAULT '{}',
    rejected   JSONB NOT NULL DEFAULT '[]',
    PRIMARY KEY (project_id, n)
);
-- Colours outside the channel palette (S9) are not a warning; drop any such
-- stored warnings so no drawing shows them.
-- The first step's idea: what the AI made of the text the Creator typed. The
-- text itself and the language live with the project in the orchestrator.
-- creator_fields lists the fields the Creator wrote, which the AI keeps when it
-- writes the idea again; input_hash and language say what it was written from.
CREATE TABLE IF NOT EXISTS project_idea (
    project_id       TEXT PRIMARY KEY,
    central_question TEXT NOT NULL DEFAULT '',
    curiosity_points JSONB NOT NULL DEFAULT '[]',
    scope            TEXT NOT NULL DEFAULT '',
    creator_fields   JSONB NOT NULL DEFAULT '[]',
    status           TEXT NOT NULL DEFAULT 'pending',
    redo_count       INTEGER NOT NULL DEFAULT 0,
    language         TEXT NOT NULL DEFAULT '',
    input_hash       TEXT NOT NULL DEFAULT '',
    approved_at      TIMESTAMPTZ,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE project_idea DROP COLUMN IF EXISTS refined_topic;
-- The AI's latest review of the idea, kept apart from the idea so dropping it
-- changes nothing about the idea. idea_hash is the hash of the idea it read.
CREATE TABLE IF NOT EXISTS project_idea_review (
    project_id TEXT PRIMARY KEY,
    review     JSONB NOT NULL,
    idea_hash  TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- The second step's brief: what the AI made of the approved idea. creator_fields
-- lists the fields the Creator wrote, which the AI keeps when it writes the brief
-- again; idea_hash says which approved idea it was written from; duration_seconds
-- is the target length, shown to the Creator as h:mm:ss.
CREATE TABLE IF NOT EXISTS project_brief (
    project_id          TEXT PRIMARY KEY,
    audience            TEXT NOT NULL DEFAULT '',
    duration_seconds    INTEGER NOT NULL DEFAULT 0,
    tone                TEXT NOT NULL DEFAULT '',
    learning_objectives JSONB NOT NULL DEFAULT '[]',
    avoid               JSONB NOT NULL DEFAULT '[]',
    creator_fields      JSONB NOT NULL DEFAULT '[]',
    status              TEXT NOT NULL DEFAULT 'pending',
    redo_count          INTEGER NOT NULL DEFAULT 0,
    language            TEXT NOT NULL DEFAULT '',
    idea_hash           TEXT NOT NULL DEFAULT '',
    approved_at         TIMESTAMPTZ,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- The AI's latest review of the brief, kept apart from the brief. brief_hash is
-- the hash of the brief it read.
CREATE TABLE IF NOT EXISTS project_brief_review (
    project_id TEXT PRIMARY KEY,
    review     JSONB NOT NULL,
    brief_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- The research step's facts, one row per fact, which the later steps read.
-- creator_fields lists the fields the Creator wrote, which a pasted answer
-- does not overwrite; the link_* columns hold the result of checking the
-- source link against the quote.
CREATE TABLE IF NOT EXISTS project_research (
    project_id      TEXT NOT NULL,
    fact_id         TEXT NOT NULL,
    question_id     TEXT NOT NULL DEFAULT '',
    claim_vi        TEXT NOT NULL DEFAULT '',
    type            TEXT NOT NULL DEFAULT 'fact',
    source_title    TEXT NOT NULL DEFAULT '',
    publisher       TEXT NOT NULL DEFAULT '',
    year            TEXT NOT NULL DEFAULT '',
    url             TEXT NOT NULL DEFAULT '',
    quote           TEXT NOT NULL DEFAULT '',
    confidence      TEXT NOT NULL DEFAULT '',
    note            TEXT NOT NULL DEFAULT '',
    must            BOOLEAN NOT NULL DEFAULT FALSE,
    keep            BOOLEAN NOT NULL DEFAULT TRUE,
    link_state      TEXT NOT NULL DEFAULT 'none',
    link_why        TEXT NOT NULL DEFAULT '',
    link_nearest    TEXT NOT NULL DEFAULT '',
    link_checked_at TEXT NOT NULL DEFAULT '',
    creator_fields  JSONB NOT NULL DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, fact_id)
);
-- The story step's structure, one row per project, which the later steps read.
-- narrative_arc is the ordered list of beats; creator_fields lists the fields
-- the Creator wrote, which a redo keeps; brief_hash and research_hash identify
-- the approved brief and the kept facts the story was written from.
CREATE TABLE IF NOT EXISTS project_story (
    project_id       TEXT PRIMARY KEY,
    central_question TEXT NOT NULL DEFAULT '',
    core_answer      TEXT NOT NULL DEFAULT '',
    narrative_arc    JSONB NOT NULL DEFAULT '[]',
    creator_fields   JSONB NOT NULL DEFAULT '[]',
    brief_hash       TEXT NOT NULL DEFAULT '',
    research_hash    TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'pending',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- The beat sheet step, which the script step reads. project_beat_sheet holds
-- what concerns the whole sheet (the gate of the step); project_beat holds the
-- beats, one row each, position giving the order the video runs them.
CREATE TABLE IF NOT EXISTS project_beat_sheet (
    project_id        TEXT PRIMARY KEY,
    next_seq          INTEGER NOT NULL DEFAULT 1,
    skeleton_approved BOOLEAN NOT NULL DEFAULT FALSE,
    rebalanced        INTEGER NOT NULL DEFAULT 0,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS project_beat (
    project_id     TEXT NOT NULL REFERENCES project_beat_sheet (project_id) ON DELETE CASCADE,
    beat_key       TEXT NOT NULL,
    position       INTEGER NOT NULL,
    fn             TEXT NOT NULL,
    title          TEXT NOT NULL DEFAULT '',
    goal           TEXT NOT NULL DEFAULT '',
    content        TEXT NOT NULL DEFAULT '',
    facts          TEXT[] NOT NULL DEFAULT '{}',
    dur            INTEGER NOT NULL,
    next           TEXT NOT NULL DEFAULT '',
    step           TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'pending',
    creator_fields TEXT[] NOT NULL DEFAULT '{}',
    redo_count     INTEGER NOT NULL DEFAULT 0,
    story_hash     TEXT NOT NULL DEFAULT '',
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, beat_key),
    UNIQUE (project_id, position)
);
-- The beats were a JSONB list on the sheet once; move them to project_beat.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = current_schema() AND table_name = 'project_beat_sheet' AND column_name = 'beats') THEN
        INSERT INTO project_beat (project_id, beat_key, position, fn, title, goal, content, facts, dur, next, step,
                                  status, creator_fields, redo_count, story_hash, updated_at)
        SELECT pbs.project_id, e.elem->>'key', e.ord::int,
               e.elem->'beat'->>'fn', COALESCE(e.elem->'beat'->>'title', ''), COALESCE(e.elem->'beat'->>'goal', ''),
               COALESCE(e.elem->'beat'->>'content', ''),
               COALESCE(ARRAY(SELECT jsonb_array_elements_text(e.elem->'beat'->'facts')), '{}'),
               (e.elem->'beat'->>'dur')::int, COALESCE(e.elem->'beat'->>'next', ''), COALESCE(e.elem->'beat'->>'step', ''),
               COALESCE(e.elem->>'status', 'pending'),
               COALESCE(ARRAY(SELECT jsonb_array_elements_text(e.elem->'creator_fields')), '{}'),
               COALESCE((e.elem->>'redo_count')::int, 0), COALESCE(e.elem->>'story_hash', ''), pbs.updated_at
        FROM project_beat_sheet pbs, jsonb_array_elements(pbs.beats) WITH ORDINALITY AS e(elem, ord)
        ON CONFLICT DO NOTHING;
        ALTER TABLE project_beat_sheet DROP COLUMN beats;
    END IF;
END $$;
-- The dialogue of the script step, one row per beat (beat_key is the key of the
-- beat in project_beat). lines is the ordered list of spoken lines; input_ref
-- and input_sha name the beat the dialogue was written from and its content
-- hash then, so a changed beat shows the dialogue as stale.
CREATE TABLE IF NOT EXISTS project_script (
    project_id     TEXT NOT NULL,
    beat_key       TEXT NOT NULL,
    lines          JSONB NOT NULL DEFAULT '[]',
    next_seq       INTEGER NOT NULL DEFAULT 1,
    status         TEXT NOT NULL DEFAULT 'pending',
    creator_fields TEXT[] NOT NULL DEFAULT '{}',
    redo_count     INTEGER NOT NULL DEFAULT 0,
    input_ref      TEXT NOT NULL DEFAULT '',
    input_sha      TEXT NOT NULL DEFAULT '',
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, beat_key)
);
-- The dialogue was a review unit once; move it to project_script.
INSERT INTO project_script (project_id, beat_key, lines, next_seq, status, creator_fields, redo_count, input_ref, input_sha, updated_at)
SELECT project_id, unit_key, COALESCE(content->'lines', '[]'::jsonb), COALESCE((content->>'next_seq')::int, 1),
       status, creator_fields, redo_count, input_ref, input_sha, updated_at
FROM review_units WHERE kind = 'script_beat'
ON CONFLICT DO NOTHING;
DELETE FROM review_units WHERE kind = 'script_beat';
-- Beats and their gate were review units once; they live in project_beat_sheet and project_beat now.
DELETE FROM review_units WHERE kind IN ('beat', 'beat_sheet');
-- The gate of the research step: whether the Creator skipped the step, the
-- number the next question code takes, and the approval of the fact table.
CREATE TABLE IF NOT EXISTS project_research_gate (
    project_id     TEXT PRIMARY KEY,
    skipped        BOOLEAN NOT NULL DEFAULT FALSE,
    next_question  INTEGER NOT NULL DEFAULT 0,
    status         TEXT NOT NULL DEFAULT 'pending',
    creator_fields TEXT[] NOT NULL DEFAULT '{}',
    redo_count     INTEGER NOT NULL DEFAULT 0,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- The lock step. project_script_review holds the review of the whole script
-- (the AI's findings, the Creator's decisions on them and the notes);
-- project_handoff the handoff note of each beat, input_ref and input_sha naming
-- the dialogue it was written from so a changed dialogue shows the note as
-- stale; project_glossary the shared glossary, one row per term in order.
CREATE TABLE IF NOT EXISTS project_script_review (
    project_id     TEXT PRIMARY KEY,
    findings       JSONB NOT NULL DEFAULT '[]',
    next_seq       INTEGER NOT NULL DEFAULT 1,
    decisions      JSONB NOT NULL DEFAULT '{}',
    ai_notes       JSONB NOT NULL DEFAULT '[]',
    my_notes       JSONB NOT NULL DEFAULT '[]',
    reviewed_sha   TEXT NOT NULL DEFAULT '',
    rounds         INTEGER NOT NULL DEFAULT 0,
    status         TEXT NOT NULL DEFAULT 'pending',
    creator_fields TEXT[] NOT NULL DEFAULT '{}',
    redo_count     INTEGER NOT NULL DEFAULT 0,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS project_handoff (
    project_id     TEXT NOT NULL,
    beat_key       TEXT NOT NULL,
    must           JSONB NOT NULL DEFAULT '[]',
    terms          JSONB NOT NULL DEFAULT '[]',
    exact          JSONB NOT NULL DEFAULT '[]',
    free           JSONB NOT NULL DEFAULT '[]',
    unsure         JSONB NOT NULL DEFAULT '[]',
    status         TEXT NOT NULL DEFAULT 'pending',
    creator_fields TEXT[] NOT NULL DEFAULT '{}',
    redo_count     INTEGER NOT NULL DEFAULT 0,
    input_ref      TEXT NOT NULL DEFAULT '',
    input_sha      TEXT NOT NULL DEFAULT '',
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, beat_key)
);
CREATE TABLE IF NOT EXISTS project_glossary (
    project_id TEXT NOT NULL,
    position   INTEGER NOT NULL,
    term       TEXT NOT NULL,
    say        TEXT NOT NULL DEFAULT '',
    draw       TEXT NOT NULL DEFAULT '',
    avoid      TEXT NOT NULL DEFAULT '',
    mine       BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, position)
);
-- The research gate and the lock step's data were review units once; move them.
INSERT INTO project_research_gate (project_id, skipped, next_question, status, creator_fields, redo_count, updated_at)
SELECT project_id, COALESCE((content->>'skipped')::boolean, FALSE), COALESCE((content->>'next_question')::int, 0),
       status, creator_fields, redo_count, updated_at
FROM review_units WHERE kind = 'research' AND unit_key = 'main'
ON CONFLICT DO NOTHING;
INSERT INTO project_script_review (project_id, findings, next_seq, decisions, ai_notes, my_notes, reviewed_sha, rounds,
                                   status, creator_fields, redo_count, updated_at)
SELECT project_id, COALESCE(content->'findings', '[]'), COALESCE((content->>'next_seq')::int, 1),
       COALESCE(content->'decisions', '{}'), COALESCE(content->'ai_notes', '[]'), COALESCE(content->'my_notes', '[]'),
       COALESCE(content->>'reviewed_sha', ''), COALESCE((content->>'rounds')::int, 0),
       status, creator_fields, redo_count, updated_at
FROM review_units WHERE kind = 'script_review' AND unit_key = 'main'
ON CONFLICT DO NOTHING;
INSERT INTO project_handoff (project_id, beat_key, must, terms, exact, free, unsure,
                             status, creator_fields, redo_count, input_ref, input_sha, updated_at)
SELECT project_id, unit_key, COALESCE(content->'must', '[]'), COALESCE(content->'terms', '[]'),
       COALESCE(content->'exact', '[]'), COALESCE(content->'free', '[]'), COALESCE(content->'unsure', '[]'),
       status, creator_fields, redo_count, input_ref, input_sha, updated_at
FROM review_units WHERE kind = 'handoff_beat'
ON CONFLICT DO NOTHING;
INSERT INTO project_glossary (project_id, position, term, say, draw, avoid, mine, updated_at)
SELECT ru.project_id, t.ord::int, COALESCE(t.elem->>'term', ''), COALESCE(t.elem->>'say', ''),
       COALESCE(t.elem->>'draw', ''), COALESCE(t.elem->>'avoid', ''), COALESCE((t.elem->>'mine')::boolean, FALSE), ru.updated_at
FROM review_units ru, jsonb_array_elements(COALESCE(ru.content->'terms', '[]')) WITH ORDINALITY AS t(elem, ord)
WHERE ru.kind = 'glossary' AND ru.unit_key = 'main'
ON CONFLICT DO NOTHING;
DELETE FROM review_units WHERE kind IN ('research', 'script_review', 'handoff_beat', 'glossary');
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
