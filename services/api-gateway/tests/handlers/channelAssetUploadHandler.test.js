'use strict';

const fs = require('fs');
const os = require('os');
const path = require('path');
const express = require('express');
const request = require('supertest');
const { channelAssetsRouter } = require('../../src/routes/channelAssets');

function buildApp(fakeClient, sharedDir, deps) {
  const app = express();
  app.use(express.json());
  app.use((req, _res, next) => {
    req.requestId = 'test-id';
    next();
  });
  app.use(channelAssetsRouter(fakeClient, sharedDir, deps));
  return app;
}

function acceptedClient() {
  return {
    request: jest.fn().mockResolvedValue({
      status: 202,
      headers: new Map(),
      body: { kind: 'intro', status: 'queued' },
    }),
  };
}

function probeReturning(result) {
  return { probeVideo: jest.fn().mockResolvedValue(result) };
}

const VALID_PROBE = { width: 1920, height: 1080, durationSeconds: 3.2 };

describe('handlers/channelAssetUploadHandler', () => {
  let sharedDir;

  beforeEach(() => {
    sharedDir = fs.mkdtempSync(path.join(os.tmpdir(), 'channel-assets-test-'));
  });

  afterEach(() => {
    fs.rmSync(sharedDir, { recursive: true, force: true });
  });

  test('valid intro upload saves the file and triggers orchestrator normalize', async () => {
    const fakeClient = acceptedClient();
    const deps = probeReturning(VALID_PROBE);
    const app = buildApp(fakeClient, sharedDir, deps);

    const res = await request(app)
      .post('/v1/channel-assets/intro')
      .attach('video', Buffer.from('fake-mp4-bytes'), { filename: 'intro.mp4', contentType: 'video/mp4' });

    expect(res.status).toBe(202);

    const expectedPath = path.join(sharedDir, 'channel-assets', 'intro', 'source.mp4');
    expect(fs.readFileSync(expectedPath).toString()).toBe('fake-mp4-bytes');
    expect(res.body.file_path).toBe(expectedPath);

    expect(fakeClient.request).toHaveBeenCalledWith(
      expect.objectContaining({
        method: 'POST',
        path: '/v1/channel-assets/intro',
        body: {
          file_path: expectedPath,
          // sha256 of "fake-mp4-bytes"
          source_hash: expect.stringMatching(/^[0-9a-f]{64}$/),
          render_quality: '1080p60',
          asset_role: 'video',
        },
      }),
    );
    expect(fakeClient.request.mock.calls[0][0].body.source_hash).toBe(res.body.source_hash);
  });

  test('render_quality from the query string overrides the 1080p60 default', async () => {
    const fakeClient = acceptedClient();
    const app = buildApp(fakeClient, sharedDir, probeReturning(VALID_PROBE));

    const res = await request(app)
      .post('/v1/channel-assets/outro?render_quality=720p30')
      .attach('video', Buffer.from('bytes'), { filename: 'outro.mp4', contentType: 'video/mp4' });

    expect(res.status).toBe(202);
    expect(fakeClient.request.mock.calls[0][0].body.render_quality).toBe('720p30');
    expect(fakeClient.request.mock.calls[0][0].path).toBe('/v1/channel-assets/outro');
  });

  test('wrong aspect ratio is rejected with a reason and no file is written', async () => {
    const fakeClient = acceptedClient();
    const app = buildApp(fakeClient, sharedDir, probeReturning({ width: 1080, height: 1080, durationSeconds: 2 }));

    const res = await request(app)
      .post('/v1/channel-assets/intro')
      .attach('video', Buffer.from('square'), { filename: 'intro.mp4', contentType: 'video/mp4' });

    expect(res.status).toBe(400);
    expect(res.body.error).toMatch(/16:9/);
    expect(res.body.error).toMatch(/1080x1080/);
    expect(fs.existsSync(path.join(sharedDir, 'channel-assets'))).toBe(false);
    expect(fakeClient.request).not.toHaveBeenCalled();
  });

  test('a clip longer than 5 seconds is rejected with a reason and no file is written', async () => {
    const fakeClient = acceptedClient();
    const app = buildApp(fakeClient, sharedDir, probeReturning({ width: 1920, height: 1080, durationSeconds: 8.4 }));

    const res = await request(app)
      .post('/v1/channel-assets/intro')
      .attach('video', Buffer.from('long'), { filename: 'intro.mp4', contentType: 'video/mp4' });

    expect(res.status).toBe(400);
    expect(res.body.error).toMatch(/at most 5 seconds/);
    expect(res.body.error).toMatch(/8\.40s/);
    expect(fs.existsSync(path.join(sharedDir, 'channel-assets'))).toBe(false);
    expect(fakeClient.request).not.toHaveBeenCalled();
  });

  test('a non-video MIME type is rejected before ffprobe is even reached', async () => {
    const fakeClient = acceptedClient();
    const deps = probeReturning(VALID_PROBE);
    const app = buildApp(fakeClient, sharedDir, deps);

    const res = await request(app)
      .post('/v1/channel-assets/intro')
      .attach('video', Buffer.from('not a video'), { filename: 'intro.gif', contentType: 'image/gif' });

    expect(res.status).toBe(400);
    expect(res.body.error).toMatch(/unsupported video type "image\/gif"/);
    expect(deps.probeVideo).not.toHaveBeenCalled();
    expect(fs.existsSync(path.join(sharedDir, 'channel-assets'))).toBe(false);
  });

  test('an unreadable video (ffprobe failure) is rejected and nothing is stored', async () => {
    const fakeClient = acceptedClient();
    const deps = { probeVideo: jest.fn().mockRejectedValue(new Error('file has no video stream')) };
    const app = buildApp(fakeClient, sharedDir, deps);

    const res = await request(app)
      .post('/v1/channel-assets/intro')
      .attach('video', Buffer.from('garbage'), { filename: 'intro.mp4', contentType: 'video/mp4' });

    expect(res.status).toBe(400);
    expect(res.body.error).toMatch(/no video stream/);
    expect(fs.existsSync(path.join(sharedDir, 'channel-assets'))).toBe(false);
    expect(fakeClient.request).not.toHaveBeenCalled();
  });

  test('intro-music upload saves the audio and triggers normalize for kind intro', async () => {
    const fakeClient = acceptedClient();
    const app = buildApp(fakeClient, sharedDir, probeReturning(VALID_PROBE));

    const res = await request(app)
      .post('/v1/channel-assets/intro-music')
      .attach('music', Buffer.from('id3-bytes'), { filename: 'sting.mp3', contentType: 'audio/mpeg' });

    expect(res.status).toBe(202);
    const expectedPath = path.join(sharedDir, 'channel-assets', 'intro', 'music.mp3');
    expect(fs.readFileSync(expectedPath).toString()).toBe('id3-bytes');
    expect(res.body.music_path).toBe(expectedPath);
    expect(fakeClient.request).toHaveBeenCalledWith(
      expect.objectContaining({
        method: 'POST',
        path: '/v1/channel-assets/intro',
        body: expect.objectContaining({
          file_path: expectedPath,
          render_quality: '1080p60',
          // FR66.5: video-assembly must see this is the music bed, not a clip.
          asset_role: 'music',
        }),
      }),
    );
  });

  test('outro-music upload writes under the outro directory', async () => {
    const fakeClient = acceptedClient();
    const app = buildApp(fakeClient, sharedDir, probeReturning(VALID_PROBE));

    const res = await request(app)
      .post('/v1/channel-assets/outro-music')
      .attach('music', Buffer.from('wav-bytes'), { filename: 'sting.wav', contentType: 'audio/wav' });

    expect(res.status).toBe(202);
    expect(fs.existsSync(path.join(sharedDir, 'channel-assets', 'outro', 'music.wav'))).toBe(true);
    expect(fakeClient.request.mock.calls[0][0].path).toBe('/v1/channel-assets/outro');
  });

  test('a non-audio MIME type is rejected and nothing is stored', async () => {
    const fakeClient = acceptedClient();
    const app = buildApp(fakeClient, sharedDir, probeReturning(VALID_PROBE));

    const res = await request(app)
      .post('/v1/channel-assets/intro-music')
      .attach('music', Buffer.from('nope'), { filename: 'sting.txt', contentType: 'text/plain' });

    expect(res.status).toBe(400);
    expect(res.body.error).toMatch(/unsupported audio type "text\/plain"/);
    expect(fs.existsSync(path.join(sharedDir, 'channel-assets'))).toBe(false);
    expect(fakeClient.request).not.toHaveBeenCalled();
  });

  test('a 4xx from orchestrator is mirrored back to the caller', async () => {
    const fakeClient = {
      request: jest.fn().mockResolvedValue({ status: 400, headers: new Map(), body: { error: 'file_path is required' } }),
    };
    const app = buildApp(fakeClient, sharedDir, probeReturning(VALID_PROBE));

    const res = await request(app)
      .post('/v1/channel-assets/intro')
      .attach('video', Buffer.from('bytes'), { filename: 'intro.mp4', contentType: 'video/mp4' });

    expect(res.status).toBe(400);
    expect(res.body).toEqual({ error: 'file_path is required' });
  });

  test('GET /v1/channel-assets/preview proxies to orchestrator', async () => {
    const fakeClient = {
      request: jest.fn().mockResolvedValue({ status: 200, headers: new Map(), body: { assets: [] } }),
    };
    const app = buildApp(fakeClient, sharedDir, probeReturning(VALID_PROBE));

    const res = await request(app).get('/v1/channel-assets/preview');

    expect(res.status).toBe(200);
    expect(res.body).toEqual({ assets: [] });
    expect(fakeClient.request).toHaveBeenCalledWith(
      expect.objectContaining({ method: 'GET', path: '/v1/channel-assets/preview' }),
    );
  });
});
