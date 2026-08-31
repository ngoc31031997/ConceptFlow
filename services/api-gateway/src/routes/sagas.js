'use strict';

const express = require('express');
const { proxyHandler } = require('../handlers/proxyHandler');

/**
 * `POST /v1/sagas/render` and `POST /v1/sagas/publish` → Orchestrator Service.
 * @param {import('../clients/httpClient').HttpClient} orchestratorClient
 */
function sagasRouter(orchestratorClient) {
  const router = express.Router();
  router.post('/v1/sagas/render', proxyHandler(orchestratorClient, 'orchestrator'));
  router.post('/v1/sagas/publish', proxyHandler(orchestratorClient, 'orchestrator'));
  return router;
}

module.exports = { sagasRouter };
