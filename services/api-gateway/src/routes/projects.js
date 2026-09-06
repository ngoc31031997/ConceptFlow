'use strict';

const express = require('express');
const multer = require('multer');
const { proxyHandler } = require('../handlers/proxyHandler');
const { videoHandler } = require('../handlers/videoHandler');
const { deleteProjectHandler } = require('../handlers/deleteProjectHandler');
const { thumbnailUploadHandler, thumbnailServeHandler } = require('../handlers/thumbnailUploadHandler');

const upload = multer({ storage: multer.memoryStorage(), limits: { fileSize: 2 * 1024 * 1024 } });

/**
 * `GET /v1/projects`, `GET /v1/projects/:id` and `POST /v1/projects/:id/retry` → Orchestrator Service.
 * `GET /v1/projects/:id/video` streams the assembled video from the shared volume.
 * `POST /v1/projects/:id/suggest-metadata` drafts SEO title/description/tags via Ollama —
 * proxied through `orchestratorAiClient` (a longer timeout than the default
 * 30s, since local LLM generation can take up to ~2 minutes).
 * `POST /v1/projects/:id/thumbnail` (multipart, field "thumbnail", max 2MB,
 * jpeg/png) saves a manually-uploaded thumbnail to the shared volume;
 * `GET /v1/projects/:id/thumbnail` serves it back for preview.
 * `DELETE /v1/projects/:id` removes the project's DB rows (Orchestrator) and its files (shared volume).
 * @param {import('../clients/httpClient').HttpClient} orchestratorClient
 * @param {string} sharedDir
 * @param {import('../clients/httpClient').HttpClient} orchestratorAiClient
 */
function projectsRouter(orchestratorClient, sharedDir, orchestratorAiClient) {
  const router = express.Router();
  router.get('/v1/projects', proxyHandler(orchestratorClient, 'orchestrator'));
  router.get('/v1/projects/:id', proxyHandler(orchestratorClient, 'orchestrator'));
  router.get('/v1/projects/:id/video', videoHandler(orchestratorClient, sharedDir));
  router.post('/v1/projects/:id/retry', proxyHandler(orchestratorClient, 'orchestrator'));
  router.post(
    '/v1/projects/:id/suggest-metadata',
    proxyHandler(orchestratorAiClient || orchestratorClient, 'orchestrator'),
  );
  router.post('/v1/projects/:id/thumbnail', upload.single('thumbnail'), thumbnailUploadHandler(sharedDir));
  router.get('/v1/projects/:id/thumbnail', thumbnailServeHandler(sharedDir));
  router.delete('/v1/projects/:id', deleteProjectHandler(orchestratorClient, sharedDir));
  return router;
}

module.exports = { projectsRouter };
