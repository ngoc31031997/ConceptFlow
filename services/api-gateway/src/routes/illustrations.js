'use strict';

const express = require('express');
const path = require('path');
const { proxyHandler } = require('../handlers/proxyHandler');
const { sharedFileHandler } = require('../handlers/sharedFileHandler');

// Same cap as rendering's backup upload limit.

const BACKUP_LIMIT = '256mb';

// A still is named by the sha-256 of what it shows.
const STILL_NAME = /^[0-9a-f]{64}\.png$/;

/**
 * The illustration library and the sample stills, served by rendering.
 *
 * Reads and folder edits use the ordinary client. Anything that renders a
 * preview (create, edit, try, rerender, and the first view of a preview image)
 * waits on the rendering service, and drawing waits on a model, so it goes through the
 * client without a timeout.
 *
 * @param {import('../clients/httpClient').HttpClient} renderingClient
 * @param {import('../clients/httpClient').HttpClient} renderingSlowClient
 * @param {string} sharedDir - absolute path the shared_artifacts volume is mounted at
 */
function illustrationsRouter(renderingClient, renderingSlowClient, sharedDir) {
  const router = express.Router();
  const fast = proxyHandler(renderingClient, 'rendering');
  const slow = proxyHandler(renderingSlowClient, 'rendering');
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
    router.delete('/v1/admin/illustrations/:id', fast);
  // Hình mẫu: making one copies and renders the drawing; undoing is a database change.
  router.post('/v1/admin/illustrations/:id/exemplar', slow);
  router.delete('/v1/admin/illustrations/:id/exemplar', fast);
  // One frame of a sample composition drawn with a kit; the PNG is then fetched by name.
  router.post('/v1/stills', slow);
  router.get(
    '/v1/stills/:file',
    sharedFileHandler(path.join(sharedDir, 'stills'), {
      param: 'file',
      namePattern: STILL_NAME,
      contentType: 'image/png',
    }),
  );
  return router;
}

module.exports = { illustrationsRouter };
