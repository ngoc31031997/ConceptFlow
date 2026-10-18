'use strict';

const express = require('express');
const path = require('path');
const { proxyHandler } = require('../handlers/proxyHandler');
const { sharedFileHandler } = require('../handlers/sharedFileHandler');

// A sample still is named by the sha-256 of what it shows.
const STILL_NAME = /^[0-9a-f]{64}\.png$/;

/**
 * The visual languages, served by authoring-service, and the sample stills the
 * rendering service writes into the shared volume.
 *
 * @param {import('../clients/httpClient').HttpClient} authoringClient
 * @param {import('../clients/httpClient').HttpClient} authoringSlowClient - rendering the samples waits on rendering
 * @param {string} sharedDir - absolute path the shared_artifacts volume is mounted at
 */
function visualLanguagesRouter(authoringClient, authoringSlowClient, sharedDir) {
  const router = express.Router();
  const fast = proxyHandler(authoringClient, 'authoring-service');
  const slow = proxyHandler(authoringSlowClient, 'authoring-service');
  router.get('/v1/visual-languages', fast);
  router.get('/v1/visual-languages/:id', fast);
  router.get('/v1/visual-languages/:id/samples', fast);
  router.get('/v1/projects/:id/visual-language', fast);
  router.post('/v1/admin/visual-languages/:id/activate', fast);
  router.post('/v1/admin/visual-languages/:id/copy', fast);
  router.put('/v1/admin/visual-languages/:id', fast);
  router.delete('/v1/admin/visual-languages/:id', fast);
  router.post('/v1/admin/visual-languages/:id/samples', slow);
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

module.exports = { visualLanguagesRouter };
