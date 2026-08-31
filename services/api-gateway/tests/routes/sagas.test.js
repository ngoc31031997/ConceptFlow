'use strict';

const express = require('express');
const request = require('supertest');
const { sagasRouter } = require('../../src/routes/sagas');

function buildApp(fakeClient) {
  const app = express();
  app.use(express.json());
  app.use((req, _res, next) => {
    req.requestId = 'test-id';
    next();
  });
  app.use(sagasRouter(fakeClient));
  return app;
}

describe('routes/sagas', () => {
  test('POST /v1/sagas/render proxies to orchestrator client', async () => {
    const fakeClient = {
      request: jest.fn().mockResolvedValue({ status: 201, headers: new Map(), body: { project_id: 'p1' } }),
    };
    const app = buildApp(fakeClient);

    const res = await request(app).post('/v1/sagas/render').send({ script: 'x' });

    expect(res.status).toBe(201);
    expect(fakeClient.request).toHaveBeenCalledWith(
      expect.objectContaining({ method: 'POST', path: '/v1/sagas/render' }),
    );
  });

  test('POST /v1/sagas/publish proxies to orchestrator client', async () => {
    const fakeClient = {
      request: jest.fn().mockResolvedValue({ status: 202, headers: new Map(), body: {} }),
    };
    const app = buildApp(fakeClient);

    const res = await request(app).post('/v1/sagas/publish').send({});

    expect(res.status).toBe(202);
    expect(fakeClient.request).toHaveBeenCalledWith(
      expect.objectContaining({ method: 'POST', path: '/v1/sagas/publish' }),
    );
  });
});
