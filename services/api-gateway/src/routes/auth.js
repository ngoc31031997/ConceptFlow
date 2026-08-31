'use strict';

const express = require('express');
const { proxyHandler } = require('../handlers/proxyHandler');

/**
 * `GET /v1/auth/youtube/start` and `GET /v1/auth/youtube/callback` → Publisher
 * Service. `/start` returns a 302 to Google which `proxyHandler` forwards
 * verbatim (not followed server-side) — see LLD Flow 2.
 * @param {import('../clients/httpClient').HttpClient} publisherClient
 */
function authRouter(publisherClient) {
  const router = express.Router();
  router.get('/v1/auth/youtube/start', proxyHandler(publisherClient, 'publisher'));
  router.get('/v1/auth/youtube/callback', proxyHandler(publisherClient, 'publisher'));
  return router;
}

module.exports = { authRouter };
