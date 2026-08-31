'use strict';

const express = require('express');
const { proxyHandler } = require('../handlers/proxyHandler');

/**
 * `GET /v1/plugins` → Content Plugin Service (interface-contracts.md routing table).
 * @param {import('../clients/httpClient').HttpClient} contentPluginClient
 */
function pluginsRouter(contentPluginClient) {
  const router = express.Router();
  router.get('/v1/plugins', proxyHandler(contentPluginClient, 'content-plugin'));
  return router;
}

module.exports = { pluginsRouter };
