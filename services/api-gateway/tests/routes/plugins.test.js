'use strict';

const express = require('express');
const request = require('supertest');
const { pluginsRouter } = require('../../src/routes/plugins');

function buildApp(fakeClient) {
  const app = express();
  app.use(express.json());
  app.use((req, _res, next) => {
    req.requestId = 'test-id';
    next();
  });
  app.use(pluginsRouter(fakeClient));
  return app;
}

describe('routes/plugins', () => {
  test('GET /v1/plugins proxies to content plugin client', async () => {
    const fakeClient = {
      request: jest.fn().mockResolvedValue({ status: 200, headers: new Map(), body: [{ id: 'p1' }] }),
    };
    const app = buildApp(fakeClient);

    const res = await request(app).get('/v1/plugins');

    expect(res.status).toBe(200);
    expect(res.body).toEqual([{ id: 'p1' }]);
    expect(fakeClient.request).toHaveBeenCalledWith(expect.objectContaining({ method: 'GET', path: '/v1/plugins' }));
  });
});
