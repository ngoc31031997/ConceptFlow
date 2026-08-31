'use strict';

const express = require('express');

/**
 * `GET /v1/progress/:id` (SSE) — handled entirely by the Gateway, not
 * proxied (interface-contracts.md, Question 3: this is the AMQP-to-SSE
 * bridge, not an HTTP passthrough).
 * @param {{ subscribe: import('express').RequestHandler }} progressHandlerInstance
 */
function progressRouter(progressHandlerInstance) {
  const router = express.Router();
  router.get('/v1/progress/:id', progressHandlerInstance.subscribe);
  return router;
}

module.exports = { progressRouter };
