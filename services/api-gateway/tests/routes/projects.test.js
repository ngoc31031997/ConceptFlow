'use strict';

const express = require('express');
const request = require('supertest');
const { projectsRouter } = require('../../src/routes/projects');

function buildApp(fakeClient) {
  const app = express();
  app.use(express.json());
  app.use((req, _res, next) => {
    req.requestId = 'test-id';
    next();
  });
  app.use(projectsRouter(fakeClient));
  return app;
}

describe('routes/projects', () => {
  test('GET /v1/projects/:id proxies to orchestrator client', async () => {
    const fakeClient = {
      request: jest.fn().mockResolvedValue({ status: 200, headers: new Map(), body: { project_id: 'p1' } }),
    };
    const app = buildApp(fakeClient);

    const res = await request(app).get('/v1/projects/p1');

    expect(res.status).toBe(200);
    expect(fakeClient.request).toHaveBeenCalledWith(
      expect.objectContaining({ method: 'GET', path: '/v1/projects/p1' }),
    );
  });

  test('POST /v1/projects/:id/retry proxies to orchestrator client', async () => {
    const fakeClient = {
      request: jest.fn().mockResolvedValue({ status: 200, headers: new Map(), body: {} }),
    };
    const app = buildApp(fakeClient);

    const res = await request(app).post('/v1/projects/p1/retry');

    expect(res.status).toBe(200);
    expect(fakeClient.request).toHaveBeenCalledWith(
      expect.objectContaining({ method: 'POST', path: '/v1/projects/p1/retry' }),
    );
  });
});
