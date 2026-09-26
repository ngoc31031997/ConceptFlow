'use strict';

const fs = require('fs');
const os = require('os');
const path = require('path');
const { deleteProjectHandler } = require('../../src/handlers/deleteProjectHandler');

function fakeRes() {
  const res = { statusCode: null, body: undefined };
  res.status = (c) => { res.statusCode = c; return res; };
  res.json = (b) => { res.body = b; return res; };
  res.end = () => res;
  return res;
}

function setup(upstream) {
  const shared = fs.mkdtempSync(path.join(os.tmpdir(), 'shared-'));
  const project = path.join(shared, 'p1');
  for (const dir of ['thumbnail', 'music', 'audio', 'video']) {
    fs.mkdirSync(path.join(project, dir), { recursive: true });
    fs.writeFileSync(path.join(project, dir, 'f'), 'x');
  }
  const client = { request: async () => upstream };
  return { shared, project, handler: deleteProjectHandler(client, shared) };
}

test('202 removes only the gateway-owned upload dirs', async () => {
  const { project, handler } = setup({ status: 202, body: { status: 'deleting' } });
  const res = fakeRes();
  await handler({ params: { id: 'p1' } }, res);
  expect(res.statusCode).toBe(202);
  expect(res.body).toEqual({ status: 'deleting' });
  expect(fs.existsSync(path.join(project, 'thumbnail'))).toBe(false);
  expect(fs.existsSync(path.join(project, 'music'))).toBe(false);
  // tts/rendering/video-assembly's directories are left for their own purge
  expect(fs.existsSync(path.join(project, 'audio'))).toBe(true);
  expect(fs.existsSync(path.join(project, 'video'))).toBe(true);
});

test('409 (a step is running) deletes nothing and is passed through', async () => {
  const { project, handler } = setup({ status: 409, body: { error: 'project is rendering; cancel it before deleting' } });
  const res = fakeRes();
  await handler({ params: { id: 'p1' } }, res);
  expect(res.statusCode).toBe(409);
  expect(fs.existsSync(path.join(project, 'thumbnail'))).toBe(true);
});
