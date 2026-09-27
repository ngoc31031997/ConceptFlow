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

  test('sends a Buffer body as raw bytes and reads a ZIP answer as bytes (CR-044 backup)', async () => {
    let capturedOptions;
    const zip = Buffer.from([0x50, 0x4b, 0x03, 0x04, 0xff]);
    const fetchImpl = async (url, opts) => {
      capturedOptions = opts;
      return {
        status: 200,
        headers: new Map([['content-type', 'application/zip']]),
        arrayBuffer: async () => zip.buffer.slice(zip.byteOffset, zip.byteOffset + zip.length),
        text: async () => { throw new Error('a ZIP must not be read as text'); },
      };
    };
    const client = createHttpClient('http://authoring:8080', { fetchImpl });
    const res = await client.request({ method: 'POST', path: '/v1/admin/illustrations/import', body: zip });
    expect(capturedOptions.body).toBe(zip);
    expect(Buffer.isBuffer(res.body)).toBe(true);
    expect(Buffer.compare(res.body, zip)).toBe(0);
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

describe('httpClient binary bodies (CR-044)', () => {
  test('keeps an image body as bytes instead of decoding it as text', async () => {
    const bytes = Uint8Array.from([0x89, 0x50, 0x4e, 0x47, 0xff, 0x00]);
    const fetchImpl = async () => ({
      status: 200,
      headers: new Map([['content-type', 'image/png']]),
      arrayBuffer: async () => bytes.buffer,
      text: async () => { throw new Error('must not be read as text'); },
    });
    const res = await createHttpClient('http://svc:8000', { fetchImpl }).request({ method: 'GET', path: '/x.png' });
    expect(Buffer.isBuffer(res.body)).toBe(true);
    expect([...res.body]).toEqual([...bytes]);
  });
});
