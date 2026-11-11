'use strict';

const express = require('express');
const { proxyHandler } = require('../handlers/proxyHandler');

/**
 * The reference library — text descriptions of a visual style shared by every
 * video — forwarded straight to visual-service:
 *
 * `GET /v1/reference-styles` — every description, in the order they were added.
 * `POST /v1/reference-styles` — add one; `PUT`/`DELETE /v1/reference-styles/:id` — edit or remove one.
 * `POST /v1/reference-styles/import` — check a pasted description and return it cleaned; stores nothing.
 * `GET /v1/reference-styles/:id/usage` — the videos that picked it.
 *
 * @param {import('../clients/httpClient').HttpClient} visualClient
 */
function referenceStylesRouter(visualClient) {
  const router = express.Router();
  const forward = proxyHandler(visualClient, 'visual-service');
  router.get('/v1/reference-styles', forward);
  router.post('/v1/reference-styles', forward);
  router.post('/v1/reference-styles/import', forward);
  router.put('/v1/reference-styles/:id', forward);
  router.delete('/v1/reference-styles/:id', forward);
  router.get('/v1/reference-styles/:id/usage', forward);
  return router;
}

module.exports = { referenceStylesRouter };
