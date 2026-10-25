'use strict';

const express = require('express');
const request = require('supertest');
const { llmCallsRouter } = require('../../src/routes/llmCalls');
const { UpstreamUnavailableError } = require('../../src/clients/httpClient');

function buildApp(client) {
  const app = express();
  app.use(express.json());
  app.use((req, _res, next) => {
    req.requestId = 'test-id';
    next();
  });
  app.use(llmCallsRouter(client));
  return app;
}

describe('call log routing', () => {
  test('stats and calls are forwarded to llm-service with their query', async () => {
    const client = { request: jest.fn().mockResolvedValue({ status: 200, headers: new Map(), body: { calls: [] } }) };
    const app = buildApp(client);

    const stats = await request(app).get('/v1/llm/stats?step=code&phase=chunk&days=30');
    const calls = await request(app).get('/v1/llm/calls?limit=50&project_id=p1');

    expect(stats.status).toBe(200);
    expect(calls.body).toEqual({ calls: [] });
    expect(client.request).toHaveBeenNthCalledWith(1, expect.objectContaining({
      method: 'GET', path: '/v1/llm/stats', query: { step: 'code', phase: 'chunk', days: '30' },
    }));
    expect(client.request).toHaveBeenNthCalledWith(2, expect.objectContaining({
      method: 'GET', path: '/v1/llm/calls', query: { limit: '50', project_id: 'p1' },
    }));
  });

  test('delete forwards the body and mirrors the answer', async () => {
    const client = { request: jest.fn().mockResolvedValue({ status: 200, headers: new Map(), body: { deleted: 2 } }) };

    const res = await request(buildApp(client)).post('/v1/llm/calls/delete').send({ ids: ['a', 'b'] });

    expect(res.status).toBe(200);
    expect(res.body).toEqual({ deleted: 2 });
    expect(client.request).toHaveBeenCalledWith(expect.objectContaining({
      method: 'POST', path: '/v1/llm/calls/delete', body: { ids: ['a', 'b'] },
    }));
  });

  test('an unreachable llm-service is a 502 naming it', async () => {
    const client = { request: jest.fn().mockRejectedValue(new UpstreamUnavailableError('down')) };

    const res = await request(buildApp(client)).get('/v1/llm/calls');

    expect(res.status).toBe(502);
    expect(res.body).toEqual({ error: 'upstream_unavailable', service: 'llm-service' });
  });
});
