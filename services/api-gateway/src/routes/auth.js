'use strict';

const express = require('express');
const { proxyHandler } = require('../handlers/proxyHandler');

/**
 * `GET /v1/auth/youtube/start`, `GET /v1/auth/youtube/callback` and
 * `GET /v1/auth/youtube/status` → Publisher Service. `/start` returns a 302
 * to Google which `proxyHandler` forwards verbatim (not followed
 * server-side) — see LLD Flow 2.
 * @param {import('../clients/httpClient').HttpClient} publisherClient
 */
function authRouter(publisherClient) {
  const router = express.Router();
  router.get('/v1/auth/youtube/start', proxyHandler(publisherClient, 'publisher'));
  router.get('/v1/auth/youtube/callback', proxyHandler(publisherClient, 'publisher'));
  router.get('/v1/auth/youtube/status', proxyHandler(publisherClient, 'publisher'));
  return router;
}

module.exports = { authRouter };
