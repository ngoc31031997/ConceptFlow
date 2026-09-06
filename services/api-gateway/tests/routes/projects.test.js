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

    test('deletes the project via orchestrator and removes its artifact directory', async () => {
      fs.mkdirSync(path.join(sharedDir, 'p1', 'video'), { recursive: true });
      fs.writeFileSync(path.join(sharedDir, 'p1', 'video', 'final.mp4'), 'x');
      const fakeClient = { request: jest.fn().mockResolvedValue({ status: 204, headers: new Map() }) };
      const app = buildApp(fakeClient, sharedDir);

      const res = await request(app).delete('/v1/projects/p1');

      expect(res.status).toBe(204);
      expect(fakeClient.request).toHaveBeenCalledWith(
        expect.objectContaining({ method: 'DELETE', path: '/v1/projects/p1' }),
      );
      expect(fs.existsSync(path.join(sharedDir, 'p1'))).toBe(false);
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
