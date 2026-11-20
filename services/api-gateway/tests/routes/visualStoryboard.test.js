'use strict';

const express = require('express');
const request = require('supertest');
const { visualStoryboardRouter } = require('../../src/routes/visualStoryboard');

const ok = () => ({
  request: jest.fn().mockResolvedValue({ status: 200, headers: new Map(), body: { ok: true } }),
});

function buildApp(plain, ai = ok()) {
  const app = express();
  app.use(express.json());
  app.use((req, _res, next) => {
    req.requestId = 'test-id';
    next();
  });
  app.use(visualStoryboardRouter(plain, ai));
  return app;
}

describe('storyboard routing', () => {
  const base = '/v1/projects/test-sb-gw/storyboard';

  test.each([
    ['get', base],
    ['put', `${base}/model`],
    ['post', `${base}/assets`],
    ['post', `${base}/assets/approve`],
    ['post', `${base}/assets/unapprove`],
    ['put', `${base}/assets/AST-HERO-001`],
    ['delete', `${base}/assets/AST-HERO-001`],
    ['post', `${base}/assets/AST-HERO-001/approve`],
    ['post', `${base}/assets/AST-HERO-001/unapprove`],
    ['delete', `${base}/units/VP-001/shots/SH-001`],
    ['post', `${base}/units/VP-001/shots/SH-001/approve`],
    ['post', `${base}/units/VP-001/shots/SH-001/unapprove`],
    ['post', `${base}/approve-all`],
    ['post', `${base}/findings/f1/apply`],
    ['post', `${base}/findings/m-asset-unused/accept`],
    ['post', `${base}/lock`],
    ['get', `${base}/versions/1`],
  ])('%s %s goes to visual-service', async (method, path) => {
    const plain = ok();
    const res = await request(buildApp(plain))[method](path).send({});
    expect(res.status).toBe(200);
    expect(plain.request).toHaveBeenCalledWith(
      expect.objectContaining({ method: method.toUpperCase(), path }),
    );
  });

  test.each([
    ['post', `${base}/assets/generate`],
    ['post', `${base}/assets/detail`],
    ['post', `${base}/assets/AST-HERO-001/detail`],
    ['post', `${base}/assets/AST-HERO-001/draw`],
    ['post', `${base}/assets/AST-HERO-001/redraw`],
    ['post', `${base}/generate`],
    ['post', `${base}/units/VP-002/generate`],
    ['post', `${base}/units/VP-002/shots`],
    ['put', `${base}/units/VP-002/shots/SH-003`],
    ['post', `${base}/units/VP-002/shots/SH-003/frame`],
    ['post', `${base}/review`],
  ])('%s %s goes to visual-service through the client with no timeout', async (method, path) => {
    const plain = ok();
    const ai = ok();
    const res = await request(buildApp(plain, ai))[method](path).set('X-External-AI', 'prompt').send({ note: 'n' });
    expect(res.status).toBe(200);
    expect(plain.request).not.toHaveBeenCalled();
    expect(ai.request).toHaveBeenCalledWith(
      expect.objectContaining({
        method: method.toUpperCase(),
        path,
        headers: expect.objectContaining({ 'x-external-ai': 'prompt' }),
      }),
    );
  });

  test.each([
    [`${base}/frames/SH-001`],
    [`${base}/assets/AST-HERO-001/preview`],
  ])('GET %s forwards the PNG with its type and cache header', async (path) => {
    const png = Buffer.from([0x89, 0x50, 0x4e, 0x47]);
    const plain = {
      request: jest.fn().mockResolvedValue({
        status: 200,
        headers: new Map([
          ['content-type', 'image/png'],
          ['cache-control', 'private, no-cache'],
        ]),
        body: png,
      }),
    };
    const res = await request(buildApp(plain)).get(path).buffer(true).parse((r, cb) => {
      const chunks = [];
      r.on('data', (c) => chunks.push(c));
      r.on('end', () => cb(null, Buffer.concat(chunks)));
    });
    expect(res.status).toBe(200);
    expect(res.headers['content-type']).toBe('image/png');
    expect(res.headers['cache-control']).toBe('private, no-cache');
    expect(Buffer.compare(res.body, png)).toBe(0);
  });

  test('the answer of visual-service is mirrored with its code', async () => {
    const plain = {
      request: jest.fn().mockResolvedValue({
        status: 409,
        headers: new Map(),
        body: { error: 'Chốt Chuỗi hình ở bước 10 trước', code: 'sequence_not_locked' },
      }),
    };
    const res = await request(buildApp(plain)).get(base);
    expect(res.status).toBe(409);
    expect(res.body.code).toBe('sequence_not_locked');
  });
});
