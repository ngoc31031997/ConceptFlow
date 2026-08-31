'use strict';

const express = require('express');
const request = require('supertest');
const { authRouter } = require('../../src/routes/auth');

function buildApp(fakeClient) {
  const app = express();
  app.use((req, _res, next) => {
    req.requestId = 'test-id';
    next();
  });
  app.use(authRouter(fakeClient));
  return app;
}

describe('routes/auth', () => {
  test('GET /v1/auth/youtube/start proxies and forwards 302 redirect verbatim', async () => {
    const headers = new Map([['location', 'https://accounts.google.com/o/oauth2']]);
    const fakeClient = { request: jest.fn().mockResolvedValue({ status: 302, headers, body: undefined }) };
    const app = buildApp(fakeClient);

    const res = await request(app).get('/v1/auth/youtube/start');

    expect(res.status).toBe(302);
    expect(res.headers.location).toBe('https://accounts.google.com/o/oauth2');
  });

  test('GET /v1/auth/youtube/callback proxies to publisher client', async () => {
    const fakeClient = {
      request: jest.fn().mockResolvedValue({ status: 200, headers: new Map(), body: { connected: true } }),
    };
    const app = buildApp(fakeClient);

    const res = await request(app).get('/v1/auth/youtube/callback?code=abc');

    expect(res.status).toBe(200);
    expect(res.body).toEqual({ connected: true });
    expect(fakeClient.request).toHaveBeenCalledWith(
      expect.objectContaining({ method: 'GET', path: '/v1/auth/youtube/callback' }),
    );
  });
});
