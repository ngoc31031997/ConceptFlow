'use strict';

/**
 * Express middleware that allows the Web GUI's browser origin to call the
 * Gateway's REST/SSE routes.
 *
 * Why: the Gateway is the GUI's entry point, but the two are served from
 * different origins (GUI on port 3000, Gateway on port 8080) — without
 * CORS headers every browser fetch/EventSource from the GUI is blocked by
 * the browser itself before it reaches this server at all.
 *
 * @param {string} allowedOrigin
 * @returns {import('express').RequestHandler}
 */
function corsMiddleware(allowedOrigin) {
  return (req, res, next) => {
    res.set('Access-Control-Allow-Origin', allowedOrigin);
    res.set('Access-Control-Allow-Methods', 'GET,POST,OPTIONS');
    res.set('Access-Control-Allow-Headers', 'Content-Type,X-Request-ID');
    if (req.method === 'OPTIONS') {
      res.status(204).end();
      return;
    }
    next();
  };
}

module.exports = { corsMiddleware };
