'use strict';

const express = require('express');
const { proxyHandler } = require('../handlers/proxyHandler');

/**
 * `GET /v1/fonts` — the fonts videos can use and the default one, forwarded
 * straight to rendering (the service that installs them).
 *
 * @param {import('../clients/httpClient').HttpClient} renderingClient
 */
function fontsRouter(renderingClient) {
  const router = express.Router();
  router.get('/v1/fonts', proxyHandler(renderingClient, 'rendering'));
  return router;
}

module.exports = { fontsRouter };
