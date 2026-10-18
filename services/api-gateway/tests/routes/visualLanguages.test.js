'use strict';

const express = require('express');
const fs = require('fs');
const os = require('os');
const path = require('path');
const request = require('supertest');
const { visualLanguagesRouter } = require('../../src/routes/visualLanguages');

const HASH = 'a'.repeat(64);
const PNG = Buffer.from('89504e470d0a1a0a0000000d49484452', 'hex');

function client(response = { status: 200, headers: new Map(), body: {} }) {
  return { request: jest.fn().mockResolvedValue(response) };
}

function buildApp(fast, slow, sharedDir) {
  const app = express();
  app.use(express.json());
  app.use((req, _res, next) => {
    req.requestId = 'test-id';
    next();
  });
  app.use(visualLanguagesRouter(fast, slow, sharedDir));
  return app;
}

describe('visual language routing', () => {
  test.each([
    ['get', '/v1/visual-languages', 'fast'],
    ['get', '/v1/visual-languages/phatsang', 'fast'],
    ['get', '/v1/visual-languages/phatsang/samples', 'fast'],
    ['get', '/v1/projects/p1/visual-language', 'fast'],
    ['post', '/v1/admin/visual-languages/phatsang/activate', 'fast'],
    ['post', '/v1/admin/visual-languages/phatsang/copy', 'fast'],
    ['put', '/v1/admin/visual-languages/phatsang', 'fast'],
    ['delete', '/v1/admin/visual-languages/phatsang', 'fast'],
    // Rendering the samples waits on the rendering service.
    ['post', '/v1/admin/visual-languages/phatsang/samples', 'slow'],
  ])('%s %s -> %s client', async (method, route, which) => {
    const fast = client();
    const slow = client();
    const res = await request(buildApp(fast, slow, os.tmpdir()))[method](route).send({});
    expect(res.status).toBe(200);
    const used = which === 'fast' ? fast : slow;
    const unused = which === 'fast' ? slow : fast;
    expect(used.request).toHaveBeenCalledTimes(1);
    expect(unused.request).not.toHaveBeenCalled();
  });
});

describe('GET /v1/stills/:file', () => {
  let dir;
  beforeEach(() => {
    dir = fs.mkdtempSync(path.join(os.tmpdir(), 'gateway-stills-'));
    fs.mkdirSync(path.join(dir, 'stills'));
    fs.writeFileSync(path.join(dir, 'stills', `${HASH}.png`), PNG);
    fs.writeFileSync(path.join(dir, 'secret.png'), PNG);
  });
  afterEach(() => fs.rmSync(dir, { recursive: true, force: true }));

  const app = () => buildApp(client(), client(), dir);

  test('serves a still with its type, size and a long cache', async () => {
    const res = await request(app()).get(`/v1/stills/${HASH}.png`);
    expect(res.status).toBe(200);
    expect(res.headers['content-type']).toBe('image/png');
    expect(res.headers['content-length']).toBe(String(PNG.length));
    expect(res.headers['cache-control']).toContain('immutable');
    expect(Buffer.compare(res.body, PNG)).toBe(0);
  });

  test('serves a byte range', async () => {
    const res = await request(app()).get(`/v1/stills/${HASH}.png`).set('Range', 'bytes=0-3');
    expect(res.status).toBe(206);
    expect(res.headers['content-range']).toBe(`bytes 0-3/${PNG.length}`);
    expect(Buffer.compare(res.body, PNG.subarray(0, 4))).toBe(0);
  });

  test('refuses a range that starts past the end', async () => {
    const res = await request(app())
      .get(`/v1/stills/${HASH}.png`)
      .set('Range', `bytes=${PNG.length + 10}-`);
    expect(res.status).toBe(416);
  });

  test('a still that is not there is 404', async () => {
    const res = await request(app()).get(`/v1/stills/${'b'.repeat(64)}.png`);
    expect(res.status).toBe(404);
  });

  test.each(['..%2Fsecret.png', 'secret.png', 'notahash.png', `${HASH}.gif`, `${HASH.toUpperCase()}.png`])(
    'only a still name is served: %s',
    async (name) => {
      const res = await request(app()).get(`/v1/stills/${name}`);
      expect(res.status).toBe(404);
    },
  );
});
