'use strict';

const express = require('express');
const fs = require('fs');
const os = require('os');
const path = require('path');
const request = require('supertest');
const { illustrationsRouter } = require('../../src/routes/illustrations');

const HASH = 'a'.repeat(64);
const PNG = Buffer.from('89504e470d0a1a0a0000000d49484452', 'hex');

function client(response = { status: 200, headers: new Map(), body: {} }) {
  return { request: jest.fn().mockResolvedValue(response) };
}

function buildApp(fast, slow, sharedDir = os.tmpdir()) {
  const app = express();
  app.use(express.json());
  app.use((req, _res, next) => {
    req.requestId = 'test-id';
    next();
  });
  app.use(illustrationsRouter(fast, slow, sharedDir));
  return app;
}

// Anything that renders a preview waits on rendering, so it must use the no-timeout client.
describe('illustration library routing', () => {
  test.each([
    ['get', '/v1/illustration-folders', 'fast'],
    ['post', '/v1/admin/illustration-folders', 'fast'],
    ['get', '/v1/illustrations', 'fast'],
    ['get', '/v1/illustrations/i1', 'fast'],
    ['post', '/v1/admin/illustrations/i1/status', 'fast'],
    ['delete', '/v1/admin/illustrations/i1', 'fast'],
    ['get', '/v1/illustrations/i1/preview.png', 'slow'],
    ['get', '/v1/illustrations/i1/preview.gif', 'slow'],
    ['post', '/v1/illustration-tries', 'slow'],
    ['post', '/v1/admin/illustrations', 'slow'],
    ['put', '/v1/admin/illustrations/i1', 'slow'],
    ['post', '/v1/admin/illustrations/i1/rerender', 'slow'],
    ['get', '/v1/illustration-style', 'fast'],
    ['post', '/v1/admin/illustrations/draw', 'slow'],
    ['post', '/v1/admin/illustrations/i1/redraw', 'slow'],
    ['get', '/v1/admin/illustrations/export', 'slow'],
    ['post', '/v1/admin/illustrations/import', 'slow'],
    // Making a Hình mẫu renders the copy; undoing it does not.
    ['post', '/v1/stills', 'slow'],
    ['post', '/v1/admin/illustrations/i1/exemplar', 'slow'],
    ['delete', '/v1/admin/illustrations/i1/exemplar', 'fast'],
  ])('%s %s -> %s client', async (method, path, which) => {
    const clients = { fast: client(), slow: client() };
    await request(buildApp(clients.fast, clients.slow))[method](path).send({});
    expect(clients[which].request).toHaveBeenCalledTimes(1);
    expect(clients[which === 'fast' ? 'slow' : 'fast'].request).not.toHaveBeenCalled();
  });

  test('a preview image passes through as bytes with its type and caching', async () => {
    const png = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x00, 0xff]);
    const headers = new Map([['content-type', 'image/png'], ['cache-control', 'private, max-age=86400']]);
    const slow = client({ status: 200, headers, body: png });
    const res = await request(buildApp(client(), slow)).get('/v1/illustrations/i1/preview.png');
    expect(res.status).toBe(200);
    expect(res.headers['content-type']).toBe('image/png');
    expect(res.headers['cache-control']).toBe('private, max-age=86400');
    expect(Buffer.compare(res.body, png)).toBe(0);
  });

  test('the library backup downloads as bytes with its file name', async () => {
    const zip = Buffer.from([0x50, 0x4b, 0x03, 0x04, 0x00, 0xff]);
    const headers = new Map([
      ['content-type', 'application/zip'],
      ['content-disposition', 'attachment; filename="conceptflow-thu-vien-hinh-20260927-1530.zip"'],
    ]);
    const slow = client({ status: 200, headers, body: zip });
    const res = await request(buildApp(client(), slow))
      .get('/v1/admin/illustrations/export')
      .buffer(true)
      .parse((r, cb) => {
        const chunks = [];
        r.on('data', (c) => chunks.push(c));
        r.on('end', () => cb(null, Buffer.concat(chunks)));
      });
    expect(res.status).toBe(200);
    expect(res.headers['content-type']).toBe('application/zip');
    expect(res.headers['content-disposition']).toContain('conceptflow-thu-vien-hinh-');
    expect(Buffer.compare(res.body, zip)).toBe(0);
  });

  test('an uploaded backup reaches rendering as the raw ZIP, with its conflict mode', async () => {
    const zip = Buffer.from([0x50, 0x4b, 0x03, 0x04, 0x00, 0xff]);
    const slow = client({ status: 200, headers: new Map(), body: { items: [] } });
    const res = await request(buildApp(client(), slow))
      .post('/v1/admin/illustrations/import?on_conflict=replace')
      .set('Content-Type', 'application/zip')
      .send(zip);
    expect(res.status).toBe(200);
    const sent = slow.request.mock.calls[0][0];
    expect(Buffer.isBuffer(sent.body)).toBe(true);
    expect(Buffer.compare(sent.body, zip)).toBe(0);
    expect(sent.query).toEqual({ on_conflict: 'replace' });
    expect(sent.headers['content-type']).toBe('application/zip');
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
