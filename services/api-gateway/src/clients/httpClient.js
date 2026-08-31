'use strict';

const DEFAULT_TIMEOUT_MS = 30_000;

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
  const timeoutMs = options.timeoutMs || DEFAULT_TIMEOUT_MS;
  const fetchImpl = options.fetchImpl || fetch;

  async function request({ method, path, headers = {}, body, query }) {
    const url = new URL(path, baseUrl);
    if (query) {
      for (const [key, value] of Object.entries(query)) {
        if (value !== undefined) url.searchParams.append(key, value);
      }
    }

    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), timeoutMs);

    try {
      const fetchOptions = {
        method,
        headers,
        signal: controller.signal,
        redirect: 'manual', // forward 3xx verbatim instead of following server-side (Flow 2)
      };
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
      throw new UpstreamUnavailableError(`Failed to reach upstream at ${baseUrl}: ${err.message}`, {
        cause: err,
      });
    } finally {
      clearTimeout(timer);
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
