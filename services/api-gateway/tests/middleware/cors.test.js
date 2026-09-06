'use strict';

const express = require('express');
const request = require('supertest');
const { corsMiddleware } = require('../../src/middleware/cors');

describe('middleware/cors', () => {
  function buildApp() {
    const app = express();
    app.use(corsMiddleware('http://localhost:3000'));
    app.get('/v1/plugins', (req, res) => res.status(200).json({ plugins: [] }));
    return app;
  }

  test('sets Access-Control-Allow-Origin to the configured GUI origin', async () => {
    const res = await request(buildApp()).get('/v1/plugins');

    expect(res.status).toBe(200);
    expect(res.headers['access-control-allow-origin']).toBe('http://localhost:3000');
  });

  test('responds 204 to an OPTIONS preflight without reaching the route', async () => {
    const res = await request(buildApp()).options('/v1/plugins');

    expect(res.status).toBe(204);
    expect(res.headers['access-control-allow-methods']).toContain('POST');
    expect(res.headers['access-control-allow-methods']).toContain('DELETE');
    expect(res.headers['access-control-allow-headers']).toContain('Content-Type');
  });
});
