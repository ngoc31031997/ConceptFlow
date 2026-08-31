'use strict';

const { UpstreamUnavailableError } = require('../clients/httpClient');

/**
 * Factory producing an Express handler that forwards a request verbatim to
 * a downstream service through the given `httpClient`, and mirrors the
 * downstream response (status + body) back verbatim.
 *
 * The Gateway deliberately does not re-validate or transform request/response
 * bodies here — that boundary decision belongs to the downstream service
 * (interface-contracts.md "Passthrough Behavior", NFR Design Question 5).
 *
 * @param {import('../clients/httpClient').HttpClient} client - bound to one
 *   downstream base URL (Orchestrator / Content Plugin / Publisher).
 * @param {string} serviceName - used only in the 502 error body, so the
 *   caller can tell which downstream failed.
 * @returns {import('express').RequestHandler}
 */
function proxyHandler(client, serviceName) {
  return async (req, res) => {
    try {
      const headers = { ...req.headers };
      // Let fetch/undici compute these for the outgoing request instead of
      // forwarding the inbound values, which describe the *inbound* socket.
      delete headers.host;
      delete headers['content-length'];
      headers['x-request-id'] = req.requestId;

      const upstreamRes = await client.request({
        method: req.method,
        path: req.path,
        headers,
        body: req.body,
        query: req.query,
      });

      // 3xx (e.g. OAuth redirect, Flow 2) must be forwarded as-is, Location included.
      const location = upstreamRes.headers && upstreamRes.headers.get && upstreamRes.headers.get('location');
      if (location) res.set('Location', location);

      if (upstreamRes.body === undefined) {
        res.status(upstreamRes.status).end();
      } else {
        res.status(upstreamRes.status).send(upstreamRes.body);
      }
    } catch (err) {
      if (err instanceof UpstreamUnavailableError) {
        res.status(502).json({ error: 'upstream_unavailable', service: serviceName });
        return;
      }
      throw err;
    }
  };
}

module.exports = { proxyHandler };
