'use strict';

const express = require('express');
const { proxyHandler } = require('../handlers/proxyHandler');

// Same cap as authoring-service's maxLibraryBackupBytes.
const BACKUP_LIMIT = '256mb';

/**
 * CR-044 — the illustration library, served by authoring-service.
 *
 * Reads and folder edits use the ordinary client. Anything that renders a
 * preview (create, edit, try, rerender, and the first view of a preview image)
 * waits on the rendering service, and drawing waits on a model, so it goes through the
 * client without a timeout.
 *
 * @param {import('../clients/httpClient').HttpClient} authoringClient
 * @param {import('../clients/httpClient').HttpClient} authoringSlowClient
 */
function illustrationsRouter(authoringClient, authoringSlowClient) {
  const router = express.Router();
  const fast = proxyHandler(authoringClient, 'authoring-service');
  const slow = proxyHandler(authoringSlowClient, 'authoring-service');
  router.get('/v1/illustration-style', fast);
  router.get('/v1/illustration-folders', fast);
  router.post('/v1/admin/illustration-folders', fast);
  router.delete('/v1/admin/illustration-folders/:id', fast);
  router.get('/v1/illustrations', fast);
  router.get('/v1/illustrations/:id', fast);
  router.get('/v1/illustrations/:id/preview.png', slow);
  router.get('/v1/illustrations/:id/preview.gif', slow);
  router.post('/v1/illustration-tries', slow);
  router.post('/v1/admin/illustrations', slow);
  router.put('/v1/admin/illustrations/:id', slow);
  router.post('/v1/admin/illustrations/:id/status', fast);
  router.post('/v1/admin/illustrations/:id/rerender', slow);
  // The AI drawer: a model call, then a check and render, up to three times.
  router.post('/v1/admin/illustrations/draw', slow);
  router.post('/v1/admin/illustrations/:id/redraw', slow);
  // Backup: the whole library as one ZIP, and restoring it — every drawing is
  // checked and rendered again, one at a time. The upload is the raw ZIP body.
  router.get('/v1/admin/illustrations/export', slow);
  router.post(
    '/v1/admin/illustrations/import',
    express.raw({ type: ['application/zip', 'application/x-zip-compressed', 'application/octet-stream'], limit: BACKUP_LIMIT }),
    slow,
  );
  // CR-052: refused (409) while a video before its result screen uses the drawing.
  router.delete('/v1/admin/illustrations/:id', fast);
  // CR-052 — Hình mẫu: making one copies and renders the drawing; undoing is a database change.
  router.post('/v1/admin/illustrations/:id/exemplar', slow);
  router.delete('/v1/admin/illustrations/:id/exemplar', fast);
  // A video's own drawing list: planning and drawing call a model.
  router.get('/v1/projects/:id/illustrations', fast);
  router.post('/v1/projects/:id/illustrations/plan', slow);
  router.post('/v1/projects/:id/illustrations/:rowId/draw', slow);
  router.post('/v1/projects/:id/illustrations/:rowId/skip', fast);
  // CR-045: delete the row's unapproved AI drawing from the library.
  router.delete('/v1/projects/:id/illustrations/:rowId/drawing', fast);
  return router;
}

module.exports = { illustrationsRouter };
