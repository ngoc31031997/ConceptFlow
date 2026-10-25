'use strict';

const express = require('express');
const { proxyHandler } = require('../handlers/proxyHandler');

/**
 * The call log of llm-service, forwarded straight to it (no transformation at
 * the Gateway boundary):
 *
 * `GET /v1/llm/stats?step=&phase=&days=` — per model, calls, failures and the
 * average cost of the successful ones (the code step's measured numbers).
 * `GET /v1/llm/calls` — the raw rows of the log, newest first.
 * `GET /v1/llm/calls/:call_id` — one row with the system and user prompt that went out.
 * `POST /v1/llm/calls/delete` — delete chosen rows, the rows older than N days,
 * or all of them.
 *
 * @param {import('../clients/httpClient').HttpClient} llmClient
 */
function llmCallsRouter(llmClient) {
  const router = express.Router();
  router.get('/v1/llm/stats', proxyHandler(llmClient, 'llm-service'));
  router.get('/v1/llm/calls', proxyHandler(llmClient, 'llm-service'));
  router.get('/v1/llm/calls/:call_id', proxyHandler(llmClient, 'llm-service'));
  router.post('/v1/llm/calls/delete', proxyHandler(llmClient, 'llm-service'));
  return router;
}

module.exports = { llmCallsRouter };
