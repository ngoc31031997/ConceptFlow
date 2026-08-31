'use strict';

const { createHttpClient, UpstreamUnavailableError } = require('../../src/clients/httpClient');

function jsonResponse(status, body, extraHeaders = {}) {
  return {
    status,
    headers: new Map([['content-type', 'application/json'], ...Object.entries(extraHeaders)]),
    text: async () => JSON.stringify(body),
  };
}

describe('httpClient', () => {
  test('forwards method/headers/body and returns status+body verbatim', async () => {
    let capturedUrl;
    let capturedOptions;
    const fetchImpl = async (url, opts) => {
      capturedUrl = url;
      capturedOptions = opts;
      return jsonResponse(200, { ok: true });
    };

    const client = createHttpClient('http://orchestrator:8000', { fetchImpl });
    const res = await client.request({
      method: 'POST',
      path: '/v1/sagas/render',
      headers: { 'X-Request-ID': 'abc-123', 'Content-Type': 'application/json' },
      body: { project_id: 'p1' },
    });

    expect(capturedUrl.toString()).toBe('http://orchestrator:8000/v1/sagas/render');
    expect(capturedOptions.method).toBe('POST');
    expect(capturedOptions.headers['X-Request-ID']).toBe('abc-123');
    expect(capturedOptions.body).toBe(JSON.stringify({ project_id: 'p1' }));
    expect(res.status).toBe(200);
    expect(res.body).toEqual({ ok: true });
  });

  test('appends query params', async () => {
    let capturedUrl;
    const fetchImpl = async (url) => {
      capturedUrl = url;
      return jsonResponse(200, {});
    };
    const client = createHttpClient('http://svc:8000', { fetchImpl });
    await client.request({ method: 'GET', path: '/v1/plugins', query: { foo: 'bar' } });
    expect(capturedUrl.searchParams.get('foo')).toBe('bar');
  });

  test('does not follow redirects (manual mode) so 302 forwards verbatim', async () => {
    let capturedOptions;
    const fetchImpl = async (url, opts) => {
      capturedOptions = opts;
      return { status: 302, headers: new Map([['location', 'https://accounts.google.com']]), text: async () => '' };
    };
    const client = createHttpClient('http://publisher:8000', { fetchImpl });
    const res = await client.request({ method: 'GET', path: '/v1/auth/youtube/start' });
    expect(capturedOptions.redirect).toBe('manual');
    expect(res.status).toBe(302);
  });

  test('throws UpstreamUnavailableError when fetch rejects (connection failure)', async () => {
    const fetchImpl = async () => {
      throw new Error('ECONNREFUSED');
    };
    const client = createHttpClient('http://down:8000', { fetchImpl });
    await expect(client.request({ method: 'GET', path: '/v1/plugins' })).rejects.toBeInstanceOf(
      UpstreamUnavailableError,
    );
  });

  test('throws UpstreamUnavailableError on timeout (AbortController)', async () => {
    const fetchImpl = (url, opts) =>
      new Promise((_resolve, reject) => {
        opts.signal.addEventListener('abort', () => {
          const err = new Error('The operation was aborted');
          err.name = 'AbortError';
          reject(err);
        });
      });
    const client = createHttpClient('http://slow:8000', { fetchImpl, timeoutMs: 10 });
    await expect(client.request({ method: 'GET', path: '/v1/plugins' })).rejects.toBeInstanceOf(
      UpstreamUnavailableError,
    );
  });
});
