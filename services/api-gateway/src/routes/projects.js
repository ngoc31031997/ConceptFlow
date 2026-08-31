'use strict';

const express = require('express');
const { proxyHandler } = require('../handlers/proxyHandler');

/**
 * `GET /v1/projects/:id` and `POST /v1/projects/:id/retry` → Orchestrator Service.
 * @param {import('../clients/httpClient').HttpClient} orchestratorClient
 */
function projectsRouter(orchestratorClient) {
  const router = express.Router();
  router.get('/v1/projects/:id', proxyHandler(orchestratorClient, 'orchestrator'));
  router.post('/v1/projects/:id/retry', proxyHandler(orchestratorClient, 'orchestrator'));
  return router;
}

module.exports = { projectsRouter };
