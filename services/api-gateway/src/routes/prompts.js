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
 * `GET /v1/admin/prompts` — list every role/language row, for the admin
 * editor screen (PromptSettingsPage).
 * `PUT /v1/admin/prompts/:role` — save an editor's wording change.
 * `POST /v1/admin/prompts/:role/reset` — CR-025 legacy. Kept so an older
 * client keeps working; after CR-027 seeding overwrites on every start, so
 * shipped wording reaches a running database on its own and nobody needs to
 * press this.
 *
 * CR-027 FR84 — the Creator-owned layer, stored apart from the shipped one:
 * `GET /v1/admin/prompt-overrides` — list the Creator's own wording.
 * `PUT /v1/admin/prompt-overrides/:role` — save it.
 * `POST /v1/admin/prompt-overrides/:role/active` — switch it on or off.
 * Switching off falls back to the shipped wording WITHOUT destroying the
 * Creator's copy, which is what the old reset endpoint could not do.
 * `DELETE /v1/admin/prompt-overrides/:role` — discard it for good.
 *
 * @param {import('../clients/httpClient').HttpClient} orchestratorClient
 */
function promptsRouter(orchestratorClient) {
  const router = express.Router();
  router.get('/v1/prompts/:role', proxyHandler(orchestratorClient, 'orchestrator'));
  router.get('/v1/projects/:projectId/prompts/:role', proxyHandler(orchestratorClient, 'orchestrator'));
  // CR-027 FR79.4 — nút "Chạy bằng AI" có gọi được gì không: web-gui hỏi
  // trước khi vẽ nút, để chỗ nào thiếu key thì giải thích chứ không hiện một
  // nút bấm vào là lỗi.
  router.get('/v1/llm/status', proxyHandler(orchestratorClient, 'orchestrator'));
  router.get('/v1/admin/prompts', proxyHandler(orchestratorClient, 'orchestrator'));
  router.put('/v1/admin/prompts/:role', proxyHandler(orchestratorClient, 'orchestrator'));
  router.post('/v1/admin/prompts/:role/reset', proxyHandler(orchestratorClient, 'orchestrator'));
  router.get('/v1/admin/prompt-overrides', proxyHandler(orchestratorClient, 'orchestrator'));
  router.put('/v1/admin/prompt-overrides/:role', proxyHandler(orchestratorClient, 'orchestrator'));
  router.post('/v1/admin/prompt-overrides/:role/active', proxyHandler(orchestratorClient, 'orchestrator'));
  router.delete('/v1/admin/prompt-overrides/:role', proxyHandler(orchestratorClient, 'orchestrator'));
  return router;
}

module.exports = { promptsRouter };
