'use strict';

const express = require('express');
const { proxyHandler } = require('../handlers/proxyHandler');
const { videoHandler } = require('../handlers/videoHandler');
const { deleteProjectHandler } = require('../handlers/deleteProjectHandler');

/**
 * `GET /v1/projects`, `GET /v1/projects/:id` and `POST /v1/projects/:id/retry` → Orchestrator Service.
 * `GET /v1/projects/:id/video` streams the assembled video from the shared volume.
 * `DELETE /v1/projects/:id` removes the project's DB rows (Orchestrator) and its files (shared volume).
 * @param {import('../clients/httpClient').HttpClient} orchestratorClient
 * @param {string} sharedDir
 */
function projectsRouter(orchestratorClient, sharedDir) {
  const router = express.Router();
  router.get('/v1/projects', proxyHandler(orchestratorClient, 'orchestrator'));
  router.get('/v1/projects/:id', proxyHandler(orchestratorClient, 'orchestrator'));
  router.get('/v1/projects/:id/video', videoHandler(orchestratorClient, sharedDir));
  router.post('/v1/projects/:id/retry', proxyHandler(orchestratorClient, 'orchestrator'));
  router.delete('/v1/projects/:id', deleteProjectHandler(orchestratorClient, sharedDir));
  return router;
}

module.exports = { projectsRouter };
