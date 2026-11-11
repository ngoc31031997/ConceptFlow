'use strict';

const express = require('express');
const request = require('supertest');
const { fontsRouter } = require('../../src/routes/fonts');
const { UpstreamUnavailableError } = require('../../src/clients/httpClient');

function buildApp(client) {
  const app = express();
  app.use((req, _res, next) => {
    req.requestId = 'test-id';
    next();
  });
  app.use(fontsRouter(client));
  return app;
}

describe('fonts routing', () => {
  test('the font list is forwarded from rendering unchanged', async () => {
    const body = { fonts: ['Be Vietnam Pro', 'Montserrat'], default: 'Be Vietnam Pro' };
    const client = { request: jest.fn().mockResolvedValue({ status: 200, headers: new Map(), body }) };

    const res = await request(buildApp(client)).get('/v1/fonts');

    expect(res.status).toBe(200);
    expect(res.body).toEqual(body);
    expect(client.request).toHaveBeenCalledWith(expect.objectContaining({ method: 'GET', path: '/v1/fonts' }));
  });

  test('a rendering error is mirrored with its status and body', async () => {
    const client = { request: jest.fn().mockResolvedValue({ status: 500, headers: new Map(), body: { error: 'font list fonts.json is malformed' } }) };

    const res = await request(buildApp(client)).get('/v1/fonts');

    expect(res.status).toBe(500);
    expect(res.body).toEqual({ error: 'font list fonts.json is malformed' });
  });

  test('an unreachable rendering is a 502 naming it', async () => {
    const client = { request: jest.fn().mockRejectedValue(new UpstreamUnavailableError('down')) };

    const res = await request(buildApp(client)).get('/v1/fonts');

    expect(res.status).toBe(502);
    expect(res.body).toEqual({ error: 'upstream_unavailable', service: 'rendering' });
  });
});
