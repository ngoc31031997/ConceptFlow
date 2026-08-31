'use strict';

const express = require('express');

/**
 * `GET /health` — always `200 {status: "ok"}`, unconditionally. Deliberately
 * does NOT check downstream service health (infrastructure-design.md /
 * NFR Design's anti-cascading-failure decision: a downstream outage must
 * not make the Gateway itself report unhealthy).
 */
function healthRouter() {
  const router = express.Router();
  router.get('/health', (req, res) => {
    res.status(200).json({ status: 'ok' });
  });
  return router;
}

module.exports = { healthRouter };
