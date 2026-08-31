'use strict';

const express = require('express');
const request = require('supertest');
const { healthRouter } = require('../../src/routes/health');

describe('routes/health', () => {
  test('GET /health returns 200 {status: "ok"} unconditionally', async () => {
    const app = express();
    app.use(healthRouter());

    const res = await request(app).get('/health');

    expect(res.status).toBe(200);
    expect(res.body).toEqual({ status: 'ok' });
  });
});
