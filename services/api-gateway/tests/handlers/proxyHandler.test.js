'use strict';

const { proxyHandler } = require('../../src/handlers/proxyHandler');
const { UpstreamUnavailableError } = require('../../src/clients/httpClient');

function makeReq(overrides = {}) {
  return {
    method: 'GET',
    path: '/v1/plugins',
    headers: { host: 'localhost:8080', 'content-length': '0' },
    query: {},
    requestId: 'req-1',
    ...overrides,
  };
}

function makeRes() {
  const res = {};
  res.statusCode = null;
  res.status = jest.fn((code) => {
    res.statusCode = code;
    return res;
  });
  res.json = jest.fn((body) => {
    res.jsonBody = body;
    return res;
  });
  res.send = jest.fn((body) => {
    res.sentBody = body;
    return res;
  });
  res.end = jest.fn(() => res);
  res.set = jest.fn(() => res);
  return res;
}

describe('proxyHandler', () => {
  test('forwards status + body verbatim on success', async () => {
    const client = { request: jest.fn().mockResolvedValue({ status: 200, headers: new Map(), body: [{ id: 1 }] }) };
    const handler = proxyHandler(client, 'content-plugin');
    const req = makeReq();
    const res = makeRes();

    await handler(req, res);

    expect(client.request).toHaveBeenCalledWith(
      expect.objectContaining({ method: 'GET', path: '/v1/plugins' }),
    );
    expect(res.status).toHaveBeenCalledWith(200);
    expect(res.send).toHaveBeenCalledWith([{ id: 1 }]);
  });

  test('forwards X-Request-ID to downstream, generated or inbound', async () => {
    const client = { request: jest.fn().mockResolvedValue({ status: 200, headers: new Map(), body: {} }) };
    const handler = proxyHandler(client, 'orchestrator');
    const req = makeReq({ requestId: 'my-correlation-id' });
    await handler(req, makeRes());

    const forwardedHeaders = client.request.mock.calls[0][0].headers;
    expect(forwardedHeaders['x-request-id']).toBe('my-correlation-id');
  });

  test('forwards 3xx redirect with Location header verbatim', async () => {
    const headers = new Map([['location', 'https://accounts.google.com/o/oauth2']]);
    const client = { request: jest.fn().mockResolvedValue({ status: 302, headers, body: undefined }) };
    const handler = proxyHandler(client, 'publisher');
    const req = makeReq({ method: 'GET', path: '/v1/auth/youtube/start' });
    const res = makeRes();

    await handler(req, res);

    expect(res.set).toHaveBeenCalledWith('Location', 'https://accounts.google.com/o/oauth2');
    expect(res.status).toHaveBeenCalledWith(302);
    expect(res.end).toHaveBeenCalled();
  });

  test('forwards downstream 4xx/5xx error bodies verbatim', async () => {
    const client = {
      request: jest.fn().mockResolvedValue({ status: 409, headers: new Map(), body: { error: 'conflict' } }),
    };
    const handler = proxyHandler(client, 'orchestrator');
    const res = makeRes();

    await handler(makeReq({ method: 'POST', path: '/v1/sagas/render' }), res);

    expect(res.status).toHaveBeenCalledWith(409);
    expect(res.send).toHaveBeenCalledWith({ error: 'conflict' });
  });

  test('maps connection failure to 502 upstream_unavailable', async () => {
    const client = { request: jest.fn().mockRejectedValue(new UpstreamUnavailableError('boom')) };
    const handler = proxyHandler(client, 'orchestrator');
    const res = makeRes();

    await handler(makeReq(), res);

    expect(res.status).toHaveBeenCalledWith(502);
    expect(res.json).toHaveBeenCalledWith({ error: 'upstream_unavailable', service: 'orchestrator' });
  });
});
