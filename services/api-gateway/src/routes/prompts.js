'use strict';

const express = require('express');
const { proxyHandler } = require('../handlers/proxyHandler');

/**
 * The prompt library and the default models belong to llm-service; the filled-in
 * prompts and the video-archetype table to authoring-service. All are proxied
 * straight through (no transformation at the Gateway boundary):
 *
 * `GET /v1/prompts/:role` — the raw template for one pipeline role (llm-service).
 * `GET /v1/projects/:projectId/prompts/:role` — the same
 * template with every {{variable}} already substituted from that project.
 * This is what web-gui's Copy button uses: the browser does not do the
 * substitution, so the copy-out path and the server's own generate call
 * cannot drift apart on the same role.
 *
 * `POST /v1/prompt-renders` — the same substitution for what the
 * browser has not saved yet (draft topic, pasted script, unapplied subtitle
 * style). web-gui does not assemble any prompt text itself.
 *
 * The prompt library (a list of prompts per role, one active):
 * `GET /v1/admin/prompts[?role=]` — list. `POST /v1/admin/prompts` — create.
 * `POST /v1/admin/prompts/:id/copy` — duplicate any row into an editable one.
 * `PUT /v1/admin/prompts/:id` — edit (system rows answer 403).
 * `POST /v1/admin/prompts/:id/activate` — make it the role's active prompt.
 * `DELETE /v1/admin/prompts/:id` — delete (system rows answer 403).
 * `GET /v1/prompts/:role` serves the role's active prompt.
 *
 * `GET /v1/settings/ai-model`, `PUT /v1/settings/ai-model/steps/:step` — the
 * default model of each step (llm-service).
 *
 * @param {import('../clients/httpClient').HttpClient} authoringClient
 * @param {import('../clients/httpClient').HttpClient} llmClient
 */
function promptsRouter(authoringClient, llmClient) {
  const router = express.Router();
  router.get('/v1/prompts/:role', proxyHandler(llmClient, 'llm-service'));
  router.get('/v1/projects/:projectId/prompts/:role', proxyHandler(authoringClient, 'authoring-service'));
  router.post('/v1/prompt-renders', proxyHandler(authoringClient, 'authoring-service'));
  // Nút "Chạy bằng AI" có gọi được gì không: web-gui hỏi
  // trước khi vẽ nút, để chỗ nào thiếu key thì giải thích chứ không hiện một
  // nút bấm vào là lỗi.
  router.get('/v1/llm/status', proxyHandler(authoringClient, 'authoring-service'));
  // Model AI mặc định của từng bước (Cài đặt): đọc cả bảng, đổi từng bước. llm-service giữ.
  router.get('/v1/settings/ai-model', proxyHandler(llmClient, 'llm-service'));
  router.put('/v1/settings/ai-model/steps/:step', proxyHandler(llmClient, 'llm-service'));
  router.get('/v1/admin/prompts', proxyHandler(llmClient, 'llm-service'));
  router.post('/v1/admin/prompts', proxyHandler(llmClient, 'llm-service'));
  router.post('/v1/admin/prompts/:id/copy', proxyHandler(llmClient, 'llm-service'));
  router.put('/v1/admin/prompts/:id', proxyHandler(llmClient, 'llm-service'));
  router.post('/v1/admin/prompts/:id/activate', proxyHandler(llmClient, 'llm-service'));
  router.delete('/v1/admin/prompts/:id', proxyHandler(llmClient, 'llm-service'));
  // The table of video kinds the Story Architect chooses from.
  router.get('/v1/video-archetypes', proxyHandler(authoringClient, 'authoring-service'));
  router.post('/v1/admin/video-archetypes', proxyHandler(authoringClient, 'authoring-service'));
  router.post('/v1/admin/video-archetypes/:id/copy', proxyHandler(authoringClient, 'authoring-service'));
  router.put('/v1/admin/video-archetypes/:id', proxyHandler(authoringClient, 'authoring-service'));
  router.delete('/v1/admin/video-archetypes/:id', proxyHandler(authoringClient, 'authoring-service'));
  return router;
}

module.exports = { promptsRouter };
