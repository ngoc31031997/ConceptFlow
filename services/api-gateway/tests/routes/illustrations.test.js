'use strict';

const express = require('express');
const request = require('supertest');
const { illustrationsRouter } = require('../../src/routes/illustrations');

function client(response = { status: 200, headers: new Map(), body: {} }) {
  return { request: jest.fn().mockResolvedValue(response) };
}

function buildApp(fast, slow) {
  const app = express();
  app.use(express.json());
  app.use((req, _res, next) => {
    req.requestId = 'test-id';
    next();
  });
  app.use(illustrationsRouter(fast, slow));
  return app;
}

// CR-044: anything that renders a preview waits on rendering, so it must use the no-timeout client.
describe('illustration library routing', () => {
  test.each([
    ['get', '/v1/illustration-folders', 'fast'],
    ['post', '/v1/admin/illustration-folders', 'fast'],
    ['get', '/v1/illustrations', 'fast'],
    ['get', '/v1/illustrations/i1', 'fast'],
    ['post', '/v1/admin/illustrations/i1/status', 'fast'],
    ['delete', '/v1/admin/illustrations/i1', 'fast'],
    ['get', '/v1/illustrations/i1/preview.png', 'slow'],
    ['get', '/v1/illustrations/i1/preview.gif', 'slow'],
    ['post', '/v1/illustration-tries', 'slow'],
    ['post', '/v1/admin/illustrations', 'slow'],
    ['put', '/v1/admin/illustrations/i1', 'slow'],
    ['post', '/v1/admin/illustrations/i1/rerender', 'slow'],
    ['get', '/v1/illustration-style', 'fast'],
    ['post', '/v1/admin/illustrations/draw', 'slow'],
    ['post', '/v1/admin/illustrations/i1/redraw', 'slow'],
  ])('%s %s -> %s client', async (method, path, which) => {
    const clients = { fast: client(), slow: client() };
    await request(buildApp(clients.fast, clients.slow))[method](path).send({});
    expect(clients[which].request).toHaveBeenCalledTimes(1);
    expect(clients[which === 'fast' ? 'slow' : 'fast'].request).not.toHaveBeenCalled();
  });

  test('a preview image passes through as bytes with its type and caching', async () => {
    const png = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x00, 0xff]);
    const headers = new Map([['content-type', 'image/png'], ['cache-control', 'private, max-age=86400']]);
    const slow = client({ status: 200, headers, body: png });
    const res = await request(buildApp(client(), slow)).get('/v1/illustrations/i1/preview.png');
    expect(res.status).toBe(200);
    expect(res.headers['content-type']).toBe('image/png');
    expect(res.headers['cache-control']).toBe('private, max-age=86400');
    expect(Buffer.compare(res.body, png)).toBe(0);
  });
});
