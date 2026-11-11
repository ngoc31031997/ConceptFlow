'use strict';

const express = require('express');
const request = require('supertest');
const { visualPlanRouter } = require('../../src/routes/visualPlan');

function buildApp(plain) {
  const app = express();
  app.use(express.json());
  app.use((req, _res, next) => {
    req.requestId = 'test-id';
    next();
  });
  app.use(visualPlanRouter(plain));
  return app;
}

const ok = () => ({ request: jest.fn().mockResolvedValue({ status: 200, headers: new Map(), body: { ok: true } }) });

describe('visual plan routing', () => {
  const base = '/v1/projects/test-plan-gw/visual-plan';

  test.each([
    ['get', base],
    ['put', `${base}/units/VP-001`],
    ['put', `${base}/units/VP-001/questions/Q1`],
    ['post', `${base}/units/VP-001/split`],
    ['post', `${base}/units/VP-001/merge`],
    ['post', `${base}/units/VP-001/approve`],
    ['post', `${base}/units/VP-001/unapprove`],
    ['post', `${base}/approve-all`],
    ['post', `${base}/findings/f1/apply`],
    ['post', `${base}/findings/m-plan-risk/accept`],
    ['post', `${base}/lock`],
    ['get', `${base}/versions/1`],
  ])('%s %s goes to visual-service', async (method, path) => {
    const plain = ok();
    const res = await request(buildApp(plain))[method](path).send({});
    expect(res.status).toBe(200);
    expect(plain.request).toHaveBeenCalledWith(expect.objectContaining({ method: method.toUpperCase(), path }));
  });

  test('the answer of visual-service is mirrored', async () => {
    const plain = { request: jest.fn().mockResolvedValue({ status: 409, headers: new Map(), body: { error: 'Chốt Ngôn ngữ hình ảnh ở bước 8 trước', code: 'concept_not_locked' } }) };
    const res = await request(buildApp(plain)).get(base);
    expect(res.status).toBe(409);
    expect(res.body.code).toBe('concept_not_locked');
  });
});
