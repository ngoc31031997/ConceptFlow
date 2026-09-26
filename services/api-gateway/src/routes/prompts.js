'use strict';

const express = require('express');
const { proxyHandler } = require('../handlers/proxyHandler');

/**
 * CR-025 — prompt-template CRUD, proxied straight through to Orchestrator
 * (same passthrough posture as sagasRouter/projectsRouter, no
 * transformation at the Gateway boundary):
 *
 * `GET /v1/prompts/:role` — the raw template for one pipeline role.
 * `GET /v1/projects/:projectId/prompts/:role` — CR-027 FR77.2, the same
 * template with every {{variable}} already substituted from that project.
 * This is what web-gui's Copy button uses: the browser no longer does the
 * substitution, so the copy-out path and the server's own generate call
 * cannot drift apart on the same role.
 *
 * `POST /v1/prompt-renders` — CR-040 FR113, the same substitution for what the
 * browser has not saved yet (draft topic, pasted script, unapplied subtitle
 * style). web-gui no longer assembles any prompt text itself.
 *
 * CR-031 — the prompt library (a list of prompts per role, one active):
 * `GET /v1/admin/prompts[?role=]` — list. `POST /v1/admin/prompts` — create.
 * `POST /v1/admin/prompts/:id/copy` — duplicate any row into an editable one.
 * `PUT /v1/admin/prompts/:id` — edit (system rows answer 403).
 * `POST /v1/admin/prompts/:id/activate` — make it the role's active prompt.
 * `DELETE /v1/admin/prompts/:id` — delete (system rows answer 403).
 * `GET /v1/prompts/:role` serves the role's active prompt.
 *
 * Since CR-040 FR111 these are served by authoring-service, not the orchestrator.
 *
 * @param {import('../clients/httpClient').HttpClient} authoringClient
 */
function promptsRouter(authoringClient) {
  const router = express.Router();
  router.get('/v1/prompts/:role', proxyHandler(authoringClient, 'authoring-service'));
  router.get('/v1/projects/:projectId/prompts/:role', proxyHandler(authoringClient, 'authoring-service'));
  router.post('/v1/prompt-renders', proxyHandler(authoringClient, 'authoring-service'));
  // CR-040 FR113 — starter scripts and hook/end-screen snippets.
  router.get('/v1/script-templates', proxyHandler(authoringClient, 'authoring-service'));
  // CR-027 FR79.4 — nút "Chạy bằng AI" có gọi được gì không: web-gui hỏi
  // trước khi vẽ nút, để chỗ nào thiếu key thì giải thích chứ không hiện một
  // nút bấm vào là lỗi.
  router.get('/v1/llm/status', proxyHandler(authoringClient, 'authoring-service'));
  router.get('/v1/admin/prompts', proxyHandler(authoringClient, 'authoring-service'));
  router.post('/v1/admin/prompts', proxyHandler(authoringClient, 'authoring-service'));
  router.post('/v1/admin/prompts/:id/copy', proxyHandler(authoringClient, 'authoring-service'));
  router.put('/v1/admin/prompts/:id', proxyHandler(authoringClient, 'authoring-service'));
  router.post('/v1/admin/prompts/:id/activate', proxyHandler(authoringClient, 'authoring-service'));
  router.delete('/v1/admin/prompts/:id', proxyHandler(authoringClient, 'authoring-service'));
  return router;
}

module.exports = { promptsRouter };
