'use strict';

const express = require('express');
const { proxyHandler } = require('../handlers/proxyHandler');

/**
 * CR-025 — prompt-template CRUD, proxied straight through to Orchestrator
 * (same passthrough posture as sagasRouter/projectsRouter, no
 * transformation at the Gateway boundary):
 *
 * `GET /v1/prompts/:role` — public read, used at runtime by web-gui's wizard
 * to fetch the current wording for one pipeline role.
 * `GET /v1/admin/prompts` — list every role/language row, for the admin
 * editor screen (PromptSettingsPage).
 * `PUT /v1/admin/prompts/:role` — save an editor's wording change.
 *
 * @param {import('../clients/httpClient').HttpClient} orchestratorClient
 */
function promptsRouter(orchestratorClient) {
  const router = express.Router();
  router.get('/v1/prompts/:role', proxyHandler(orchestratorClient, 'orchestrator'));
  router.get('/v1/admin/prompts', proxyHandler(orchestratorClient, 'orchestrator'));
  router.put('/v1/admin/prompts/:role', proxyHandler(orchestratorClient, 'orchestrator'));
  return router;
}

module.exports = { promptsRouter };
