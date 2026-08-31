'use strict';

const { v4: uuidv4 } = require('uuid');

/**
 * Express middleware that ensures every request carries an `X-Request-ID`.
 *
 * Why: the Gateway is the system's entry point (interface-contracts.md,
 * "Correlation ID Propagation") — if the GUI didn't set one, we must
 * generate it here so every downstream service and every log line can be
 * correlated back to a single inbound request.
 *
 * @returns {import('express').RequestHandler}
 */
function correlationMiddleware() {
  return (req, res, next) => {
    const incoming = req.get('X-Request-ID');
    const requestId = incoming && incoming.trim() !== '' ? incoming : uuidv4();
    req.requestId = requestId;
    res.set('X-Request-ID', requestId);
    next();
  };
}

module.exports = { correlationMiddleware };
