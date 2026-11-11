'use strict';

const express = require('express');
const request = require('supertest');
const { referenceStylesRouter } = require('../../src/routes/referenceStyles');
const { UpstreamUnavailableError } = require('../../src/clients/httpClient');

function buildApp(client) {
  const app = express();
  app.use(express.json());
  app.use((req, _res, next) => {
    req.requestId = 'test-id';
    next();
  });
  app.use(referenceStylesRouter(client));
  return app;
}

function answering(status, body) {
  return { request: jest.fn().mockResolvedValue({ status, headers: new Map(), body }) };
}

describe('reference library routing', () => {
  test('list and usage are forwarded to visual-service', async () => {
    const client = answering(200, { styles: [] });
    const app = buildApp(client);

    const list = await request(app).get('/v1/reference-styles');
    await request(app).get('/v1/reference-styles/R01/usage');

    expect(list.body).toEqual({ styles: [] });
    expect(client.request).toHaveBeenNthCalledWith(1, expect.objectContaining({ method: 'GET', path: '/v1/reference-styles' }));
    expect(client.request).toHaveBeenNthCalledWith(2, expect.objectContaining({ method: 'GET', path: '/v1/reference-styles/R01/usage' }));
  });

  test('create and edit forward the body and mirror the status', async () => {
    const client = answering(201, { id: 'R02' });
    const app = buildApp(client);
    const body = { name: 'n', summary: 's', tags: ['x'] };

    const created = await request(app).post('/v1/reference-styles').send(body);
    await request(app).put('/v1/reference-styles/R02').send(body);

    expect(created.status).toBe(201);
    expect(client.request).toHaveBeenNthCalledWith(1, expect.objectContaining({ method: 'POST', path: '/v1/reference-styles', body }));
    expect(client.request).toHaveBeenNthCalledWith(2, expect.objectContaining({ method: 'PUT', path: '/v1/reference-styles/R02', body }));
  });

  test('import forwards the pasted JSON and mirrors a 400 with its reason', async () => {
    const client = answering(400, { error: 'drawing.shape: dài quá 300 ký tự' });
    const body = { name: 'n', summary: 's', drawing: { shape: 'x' } };

    const res = await request(buildApp(client)).post('/v1/reference-styles/import').send(body);

    expect(res.status).toBe(400);
    expect(res.body).toEqual({ error: 'drawing.shape: dài quá 300 ký tự' });
    expect(client.request).toHaveBeenCalledWith(expect.objectContaining({ method: 'POST', path: '/v1/reference-styles/import', body }));
  });

  test('delete mirrors 204 and a 404 error', async () => {
    const ok = await request(buildApp(answering(204, null))).delete('/v1/reference-styles/R02');
    const missing = await request(buildApp(answering(404, { error: 'không có' }))).delete('/v1/reference-styles/R99');

    expect(ok.status).toBe(204);
    expect(missing.status).toBe(404);
    expect(missing.body).toEqual({ error: 'không có' });
  });

  test('an unreachable visual-service is a 502 naming it', async () => {
    const client = { request: jest.fn().mockRejectedValue(new UpstreamUnavailableError('down')) };

    const res = await request(buildApp(client)).get('/v1/reference-styles');

    expect(res.status).toBe(502);
    expect(res.body).toEqual({ error: 'upstream_unavailable', service: 'visual-service' });
  });
});
