'use strict';

const { Agent } = require('undici');

const DEFAULT_TIMEOUT_MS = 30_000;

// Node's built-in fetch gives up after 300s without response headers
// (UND_ERR_HEADERS_TIMEOUT) no matter what AbortController does. A
// timeoutMs of 0 promises "wait indefinitely", so that client must switch
// undici's own timeouts off too — otherwise a long LLM call is reported as
// 502 while the orchestrator keeps running it and holds its per-step lock.
const NO_TIMEOUT_DISPATCHER = new Agent({ headersTimeout: 0, bodyTimeout: 0 });

/**
 * Error thrown when the downstream service could not be reached at all
 * (connection refused, DNS failure, or the 30s timeout firing) — as opposed
 * to the downstream service responding with a 4xx/5xx, which is a normal
 * response the handler forwards verbatim. `proxyHandler` maps only this
 * error type to the Gateway's own 502 `upstream_unavailable` contract
 * (interface-contracts.md "Error Contract").
 */
class UpstreamUnavailableError extends Error {
  constructor(message, { cause } = {}) {
    super(message);
    this.name = 'UpstreamUnavailableError';
    this.cause = cause;
  }
}

/**
 * Creates a thin wrapper over the built-in `fetch` bound to a single
 * downstream base URL (one instance per downstream service — see
 * dependency-injection.md). Kept deliberately dumb: no retries, no
 * validation, no transformation — the Gateway forwards requests verbatim
 * and lets the downstream service own its own contract
 * (NFR Design Question 5).
 *
 * @param {string} baseUrl - e.g. "http://orchestrator:8000"
 * @param {{ timeoutMs?: number, fetchImpl?: typeof fetch }} [options]
 * @returns {{ request: (req: {
 *   method: string,
 *   path: string,
 *   headers?: Record<string, string>,
 *   body?: any,
 *   query?: Record<string, string>,
 * }) => Promise<{ status: number, headers: Headers, body: any, redirected: boolean }> }}
 */
function createHttpClient(baseUrl, options = {}) {
  // timeoutMs: 0 means "no timeout" (wait for upstream indefinitely).
  const timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;
  const fetchImpl = options.fetchImpl || fetch;

  async function request({ method, path, headers = {}, body, query }) {
    const url = new URL(path, baseUrl);
    if (query) {
      for (const [key, value] of Object.entries(query)) {
        if (value !== undefined) url.searchParams.append(key, value);
      }
    }

    const controller = new AbortController();
    const timer = timeoutMs > 0 ? setTimeout(() => controller.abort(), timeoutMs) : null;

    try {
      const fetchOptions = {
        method,
        headers,
        signal: controller.signal,
        redirect: 'manual', // forward 3xx verbatim instead of following server-side (Flow 2)
      };
      if (timeoutMs === 0) fetchOptions.dispatcher = NO_TIMEOUT_DISPATCHER;
      if (body !== undefined && body !== null && method !== 'GET' && method !== 'HEAD') {
        fetchOptions.body = typeof body === 'string' ? body : JSON.stringify(body);
      }

      const res = await fetchImpl(url, fetchOptions);
      const responseBody = await readResponseBody(res);

      return {
        status: res.status,
        headers: res.headers,
        body: responseBody,
      };
    } catch (err) {
      if (err.name === 'UpstreamUnavailableError') throw err;
      // fetch reports every network failure as "fetch failed"; the real reason
      // (timeout, refused, reset) is only on err.cause.
      const reason = err.cause && (err.cause.code || err.cause.message);
      const detail = reason ? `${err.message} (${reason})` : err.message;
      throw new UpstreamUnavailableError(`Failed to reach upstream at ${baseUrl}: ${detail}`, {
        cause: err,
      });
    } finally {
      if (timer) clearTimeout(timer);
    }
  }

  return { request };
}

/**
 * Reads the response body, preserving JSON as parsed objects (needed so
 * `proxyHandler` can hand it straight to Express's `res.json`) while
 * falling back to raw text for non-JSON or empty bodies.
 */
async function readResponseBody(res) {
  const text = await res.text();
  if (text === '') return undefined;
  const contentType = res.headers.get('content-type') || '';
  if (contentType.includes('application/json')) {
    try {
      return JSON.parse(text);
    } catch {
      return text;
    }
  }
  return text;
}

module.exports = { createHttpClient, UpstreamUnavailableError };
