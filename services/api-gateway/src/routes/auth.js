'use strict';

const express = require('express');
const { proxyHandler } = require('../handlers/proxyHandler');

/**
 * YouTube OAuth routes → Publisher Service. `/start` returns a 302 to Google
 * which `proxyHandler` forwards verbatim (not followed server-side) — see
 * LLD Flow 2.
 *
 * CR-012 added the app catalogue and the per-channel account routes. They are
 * listed individually rather than mounted as a wildcard so the Gateway keeps
 * stating exactly which downstream surface it exposes (interface-contracts.md).
 *
 * @param {import('../clients/httpClient').HttpClient} publisherClient
 */
function authRouter(publisherClient) {
  const router = express.Router();
  const toPublisher = proxyHandler(publisherClient, 'publisher');

  router.get('/v1/auth/youtube/start', toPublisher);
  router.get('/v1/auth/youtube/callback', toPublisher);
  router.get('/v1/auth/youtube/status', toPublisher);
  // CR-012: the OAuth clients on offer, and the channels connected through them.
  router.get('/v1/auth/youtube/apps', toPublisher);
  router.get('/v1/auth/youtube/accounts', toPublisher);
  router.delete('/v1/auth/youtube/accounts/:channelId', toPublisher);
  router.post('/v1/auth/youtube/accounts/:channelId/default', toPublisher);
  return router;
}

module.exports = { authRouter };
