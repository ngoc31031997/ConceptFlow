'use strict';

const express = require('express');
const request = require('supertest');
const { visualConceptRouter } = require('../../src/routes/visualConcept');

function buildApp(plain, ai) {
  const app = express();
  app.use(express.json());
  app.use((req, _res, next) => {
    req.requestId = 'test-id';
    next();
  });
  app.use(visualConceptRouter(plain, ai));
  return app;
}

const ok = () => ({ request: jest.fn().mockResolvedValue({ status: 200, headers: new Map(), body: { ok: true } }) });

describe('visual concept routing', () => {
  const base = '/v1/projects/test-cr112b-gw/visual-concept';

  test.each([
    ['get', base],
    ['put', `${base}/refs`],
    ['put', `${base}/model`],
    ['put', `${base}/units/typography`],
    ['post', `${base}/units/typography/approve`],
    ['post', `${base}/units/typography/unapprove`],
    ['post', `${base}/approve-all`],
    ['post', `${base}/findings/f1/apply`],
    ['post', `${base}/findings/m-terms/accept`],
    ['post', `${base}/lock`],
    ['get', `${base}/versions/1`],
  ])('%s %s goes to visual-service', async (method, path) => {
    const plain = ok();
    const ai = ok();
    const res = await request(buildApp(plain, ai))[method](path).send({});
    expect(res.status).toBe(200);
    expect(plain.request).toHaveBeenCalledWith(expect.objectContaining({ method: method.toUpperCase(), path }));
    expect(ai.request).not.toHaveBeenCalled();
  });

  test.each([`${base}/generate`, `${base}/units/color_system/generate`, `${base}/review`])(
    '%s asks the AI and waits on it, with the external-AI header',
    async (path) => {
      const plain = ok();
      const ai = ok();
      const res = await request(buildApp(plain, ai)).post(path).set('X-External-AI', 'prompt').send({ note: 'n' });
      expect(res.status).toBe(200);
      expect(plain.request).not.toHaveBeenCalled();
      expect(ai.request).toHaveBeenCalledWith(expect.objectContaining({
        method: 'POST', path, body: { note: 'n' }, headers: expect.objectContaining({ 'x-external-ai': 'prompt' }),
      }));
    },
  );

  test('the answer of visual-service is mirrored', async () => {
    const plain = { request: jest.fn().mockResolvedValue({ status: 409, headers: new Map(), body: { error: 'Chốt kịch bản ở bước 7 trước', code: 'script_not_locked' } }) };
    const res = await request(buildApp(plain, ok())).get(base);
    expect(res.status).toBe(409);
    expect(res.body.code).toBe('script_not_locked');
  });
});
