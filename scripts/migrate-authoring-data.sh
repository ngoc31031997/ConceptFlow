#!/usr/bin/env bash
# CR-040 FR111 — one-time copy of authoring data from the orchestrator's database
# into authoring-service's own database. Run once, after `docker compose up -d`
# has started authoring-service (so its schema and the seeded system prompts
# exist) and before the old rows are dropped from the orchestrator database.
#
# Copies (idempotent — rerunning skips what is already there):
#   project_authoring  every row, plus `language` and `created_at` taken from
#                      the project (they now live beside the topic)
#   llm_usage          every row, keeping its timestamp
#   prompts            the Creator's own prompts (is_system = false), and which
#                      prompt was active per role. System prompts are not copied:
#                      authoring-service seeds its own on start.
#
# Nothing is deleted from the orchestrator database. Once you have checked the
# result, the old tables can be dropped by hand.
set -euo pipefail

cd "$(dirname "$0")/.."
: "${POSTGRES_USER:?set POSTGRES_USER (same value as in .env)}"

src() { docker compose exec -T orchestrator-db psql -U "$POSTGRES_USER" -d orchestrator -v ON_ERROR_STOP=1 "$@"; }
dst() { docker compose exec -T authoring-service-db psql -U "$POSTGRES_USER" -d authoring -v ON_ERROR_STOP=1 "$@"; }

echo "== project_authoring"
{
  echo "CREATE TEMP TABLE _pa (LIKE project_authoring INCLUDING DEFAULTS);"
  echo "COPY _pa (project_id, story_content, storyboard_content, code_content, review_content, topic, authoring_mode, story_model, storyboard_model, code_model, language, created_at, updated_at) FROM STDIN;"
  src -c "COPY (
      SELECT a.project_id, a.story_content, a.storyboard_content, a.code_content, a.review_content, a.topic,
             a.authoring_mode, a.story_model, a.storyboard_model, a.code_model,
             COALESCE(p.voice_language, ''), COALESCE(p.created_at, a.updated_at), a.updated_at
      FROM project_authoring a LEFT JOIN projects p ON p.project_id = a.project_id
    ) TO STDOUT"
  echo '\.'
  echo "INSERT INTO project_authoring SELECT * FROM _pa ON CONFLICT (project_id) DO NOTHING;"
} | dst

echo "== llm_usage"
COUNT=$(dst -tA -c "SELECT count(*) FROM llm_usage")
if [ "$COUNT" = "0" ]; then
  {
    echo "COPY llm_usage (created_at, provider, model, role, step, project_id, prompt_tokens, completion_tokens, reasoning_tokens, cached_tokens, duration_ms, ok, error_kind, phase) FROM STDIN;"
    src -c "COPY (SELECT created_at, provider, model, role, step, project_id, prompt_tokens, completion_tokens, reasoning_tokens, cached_tokens, duration_ms, ok, error_kind, phase FROM llm_usage ORDER BY id) TO STDOUT"
    echo '\.'
  } | dst
else
  echo "llm_usage already has $COUNT rows in authoring-service — skipped"
fi

echo "== prompts (Creator's own)"
{
  echo "CREATE TEMP TABLE _p (id text, role text, name text, template_text text, is_active boolean, created_at timestamptz, updated_at timestamptz);"
  echo "COPY _p FROM STDIN;"
  src -c "COPY (SELECT id, role, name, template_text, is_active, created_at, updated_at FROM prompts WHERE NOT is_system) TO STDOUT"
  echo '\.'
  echo "INSERT INTO prompts (id, role, name, template_text, is_system, is_active, created_at, updated_at)
        SELECT id, role, name, template_text, false, false, created_at, updated_at FROM _p ON CONFLICT (id) DO NOTHING;"
  # Restore the active prompt per role: deactivate the seeded system one first,
  # since the database refuses two active rows for a role.
  echo "UPDATE prompts SET is_active = false WHERE role IN (SELECT role FROM _p WHERE is_active) AND is_system;"
  echo "UPDATE prompts SET is_active = true WHERE id IN (SELECT id FROM _p WHERE is_active);"
} | dst

echo "== done"
dst -c "SELECT (SELECT count(*) FROM project_authoring) AS project_authoring, (SELECT count(*) FROM llm_usage) AS llm_usage, (SELECT count(*) FROM prompts WHERE NOT is_system) AS creator_prompts"
