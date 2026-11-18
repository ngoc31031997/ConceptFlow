'use strict';

const express = require('express');
const request = require('supertest');
const { visualSequenceRouter } = require('../../src/routes/visualSequence');

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
  app.use(visualSequenceRouter(plain, ai));
  return app;
}


describe('visual sequence routing', () => {
  const base = '/v1/projects/test-seq-gw/visual-sequence';

  test.each([
    ['get', base],
    ['put', `${base}/units/VP-001`],
    ['post', `${base}/units/VP-001/approve`],
    ['post', `${base}/units/VP-001/unapprove`],
    ['post', `${base}/approve-all`],
    ['post', `${base}/beats/b2/confirm`],
    ['post', `${base}/motifs`],
    ['post', `${base}/motifs/approve`],
    ['post', `${base}/motifs/unapprove`],
    ['put', `${base}/motifs/MOTIF-01`],
    ['delete', `${base}/motifs/MOTIF-01`],
    ['post', `${base}/findings/f1/apply`],
    ['post', `${base}/findings/m-seq-via/accept`],
    ['post', `${base}/lock`],
    ['get', `${base}/versions/1`],
    ['put', `${base}/model`],
  ])('%s %s goes to visual-service', async (method, path) => {
    const plain = ok();
    const res = await request(buildApp(plain))[method](path).send({});
    expect(res.status).toBe(200);
    expect(plain.request).toHaveBeenCalledWith(
      expect.objectContaining({ method: method.toUpperCase(), path }),
    );
  });

  test.each([
    ['post', `${base}/motifs/generate`],
    ['post', `${base}/generate`],
    ['post', `${base}/beats/b2/generate`],
    ['post', `${base}/units/VP-002/generate`],
    ['post', `${base}/review`],
  ])('%s %s goes to visual-service through the client with no timeout', async (method, path) => {
    const plain = ok();
    const ai = ok();
    const res = await request(buildApp(plain, ai))[method](path).set('X-External-AI', 'prompt').send({ note: 'n' });
    expect(res.status).toBe(200);
    expect(plain.request).not.toHaveBeenCalled();
    expect(ai.request).toHaveBeenCalledWith(expect.objectContaining({ method: 'POST', path }));
  });

  test('the answer of visual-service is mirrored', async () => {
    const plain = {
      request: jest.fn().mockResolvedValue({
        status: 409,
        headers: new Map(),
        body: { error: 'Chốt Kế hoạch hình ở bước 9 trước', code: 'plan_not_locked' },
      }),
    };
    const res = await request(buildApp(plain)).get(base);
    expect(res.status).toBe(409);
    expect(res.body.code).toBe('plan_not_locked');
  });
});
