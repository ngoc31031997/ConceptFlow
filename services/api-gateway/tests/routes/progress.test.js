'use strict';

const express = require('express');
const request = require('supertest');
const { progressRouter } = require('../../src/routes/progress');

describe('routes/progress', () => {
  test('GET /v1/progress/:id delegates to progressHandler.subscribe', async () => {
    const subscribe = jest.fn((req, res) => {
      res.status(200).set('Content-Type', 'text/event-stream').end();
    });
    const app = express();
    app.use(progressRouter({ subscribe }));

    const res = await request(app).get('/v1/progress/proj-1');

    expect(subscribe).toHaveBeenCalled();
    expect(res.status).toBe(200);
    expect(res.headers['content-type']).toContain('text/event-stream');
  });
});
