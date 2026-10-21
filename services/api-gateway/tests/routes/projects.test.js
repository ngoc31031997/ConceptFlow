'use strict';

const fs = require('fs');
const os = require('os');
const path = require('path');
const express = require('express');
const request = require('supertest');
const { projectsRouter } = require('../../src/routes/projects');

function buildApp(fakeClient, sharedDir) {
  const app = express();
  app.use(express.json());
  app.use((req, _res, next) => {
    req.requestId = 'test-id';
    next();
  });
  app.use(projectsRouter(fakeClient, sharedDir));
  return app;
}

describe('routes/projects', () => {
  test('GET /v1/projects/:id proxies to orchestrator client', async () => {
    const fakeClient = {
      request: jest.fn().mockResolvedValue({ status: 200, headers: new Map(), body: { project_id: 'p1' } }),
    };
    const app = buildApp(fakeClient);

    const res = await request(app).get('/v1/projects/p1');

    expect(res.status).toBe(200);
    expect(fakeClient.request).toHaveBeenCalledWith(
      expect.objectContaining({ method: 'GET', path: '/v1/projects/p1' }),
    );
  });

  test('POST /v1/projects/:id/retry proxies to orchestrator client', async () => {
    const fakeClient = {
      request: jest.fn().mockResolvedValue({ status: 200, headers: new Map(), body: {} }),
    };
    const app = buildApp(fakeClient);

    const res = await request(app).post('/v1/projects/p1/retry');

    expect(res.status).toBe(200);
    expect(fakeClient.request).toHaveBeenCalledWith(
      expect.objectContaining({ method: 'POST', path: '/v1/projects/p1/retry' }),
    );
  });

  test('GET /v1/projects/:id/qc-report proxies to orchestrator client', async () => {
    const fakeClient = {
      request: jest.fn().mockResolvedValue({
        status: 200,
        headers: new Map(),
        body: { project_id: 'p1', status: 'has_findings', findings: [] },
      }),
    };
    const app = buildApp(fakeClient);

    const res = await request(app).get('/v1/projects/p1/qc-report');

    expect(res.status).toBe(200);
    expect(fakeClient.request).toHaveBeenCalledWith(
      expect.objectContaining({ method: 'GET', path: '/v1/projects/p1/qc-report' }),
    );
  });

  test('GET /v1/projects proxies to orchestrator client', async () => {
    const fakeClient = {
      request: jest.fn().mockResolvedValue({ status: 200, headers: new Map(), body: { projects: [] } }),
    };
    const app = buildApp(fakeClient);

    const res = await request(app).get('/v1/projects');

    expect(res.status).toBe(200);
    expect(fakeClient.request).toHaveBeenCalledWith(
      expect.objectContaining({ method: 'GET', path: '/v1/projects' }),
    );
  });

  describe('DELETE /v1/projects/:id', () => {
    let sharedDir;

    beforeEach(() => {
      sharedDir = fs.mkdtempSync(path.join(os.tmpdir(), 'shared-'));
    });

    afterEach(() => {
      fs.rmSync(sharedDir, { recursive: true, force: true });
    });

    test('deletes via orchestrator and removes only the gateway-owned upload dirs', async () => {
      fs.mkdirSync(path.join(sharedDir, 'p1', 'video'), { recursive: true });
      fs.writeFileSync(path.join(sharedDir, 'p1', 'video', 'final.mp4'), 'x');
      fs.mkdirSync(path.join(sharedDir, 'p1', 'thumbnail'), { recursive: true });
      const fakeClient = { request: jest.fn().mockResolvedValue({ status: 202, headers: new Map(), body: { status: 'deleting' } }) };
      const app = buildApp(fakeClient, sharedDir);

      const res = await request(app).delete('/v1/projects/p1');

      expect(res.status).toBe(202);
      expect(fakeClient.request).toHaveBeenCalledWith(
        expect.objectContaining({ method: 'DELETE', path: '/v1/projects/p1' }),
      );
      expect(fs.existsSync(path.join(sharedDir, 'p1', 'thumbnail'))).toBe(false);
      // tts/rendering/video-assembly purge their own files
      expect(fs.existsSync(path.join(sharedDir, 'p1', 'video', 'final.mp4'))).toBe(true);
    });

    test('does not touch the filesystem when orchestrator returns 404', async () => {
      fs.mkdirSync(path.join(sharedDir, 'p1'), { recursive: true });
      const fakeClient = {
        request: jest.fn().mockResolvedValue({ status: 404, headers: new Map(), body: { error: 'not_found' } }),
      };
      const app = buildApp(fakeClient, sharedDir);

      const res = await request(app).delete('/v1/projects/p1');

      expect(res.status).toBe(404);
      expect(fs.existsSync(path.join(sharedDir, 'p1'))).toBe(true);
    });

    test('rejects a path-traversal project id without touching the filesystem', async () => {
      const fakeClient = { request: jest.fn().mockResolvedValue({ status: 204, headers: new Map() }) };
      const app = buildApp(fakeClient, sharedDir);

      const res = await request(app).delete('/v1/projects/..%2F..%2Fetc');

      expect(res.status).toBe(204);
      expect(fs.existsSync(sharedDir)).toBe(true);
    });
  });
});

describe('GET /v1/projects/:id/audio/:file', () => {
  const WAV = Buffer.from('RIFF0000WAVEfmt ', 'ascii');
  const NAME = 'b1-2_0123456789ab.wav';
  let dir;
  beforeEach(() => {
    dir = fs.mkdtempSync(path.join(os.tmpdir(), 'gateway-audio-'));
    fs.mkdirSync(path.join(dir, 'p1', 'audio'), { recursive: true });
    fs.writeFileSync(path.join(dir, 'p1', 'audio', NAME), WAV);
    fs.writeFileSync(path.join(dir, 'p1', 'audio', '0_vi-VN-HoaiMyNeural_0123456789ab.wav'), WAV);
    fs.mkdirSync(path.join(dir, 'p2', 'audio'), { recursive: true });
    fs.writeFileSync(path.join(dir, 'p2', 'audio', 'b9-9_0123456789ab.wav'), WAV);
  });
  afterEach(() => fs.rmSync(dir, { recursive: true, force: true }));

  test('serves a line file of the project as audio/wav', async () => {
    const res = await request(buildApp({ request: jest.fn() }, dir)).get(`/v1/projects/p1/audio/${NAME}`);
    expect(res.status).toBe(200);
    expect(res.headers['content-type']).toMatch(/audio\/wav/);
    expect(res.headers['accept-ranges']).toBe('bytes');
  });

  test('supports a Range request so the player can seek', async () => {
    const res = await request(buildApp({ request: jest.fn() }, dir))
      .get(`/v1/projects/p1/audio/${NAME}`)
      .set('Range', 'bytes=0-3');
    expect(res.status).toBe(206);
    expect(res.headers['content-range']).toBe(`bytes 0-3/${WAV.length}`);
  });

  test.each([
    ['a scene file that is not a line file', '0_vi-VN-HoaiMyNeural_0123456789ab.wav'],
    ['a name that tries to leave the folder', '..%2F..%2Fsecret.wav'],
    ['a line file of another project', 'b9-9_0123456789ab.wav'],
    ['a file that does not exist', 'b7-7_0123456789ab.wav'],
  ])('404 for %s', async (_label, file) => {
    const res = await request(buildApp({ request: jest.fn() }, dir)).get(`/v1/projects/p1/audio/${file}`);
    expect(res.status).toBe(404);
  });

  test('404 for a project id that could point outside the shared folder', async () => {
    const res = await request(buildApp({ request: jest.fn() }, dir)).get(`/v1/projects/..%2Fp2/audio/b9-9_0123456789ab.wav`);
    expect(res.status).toBe(404);
  });
});
