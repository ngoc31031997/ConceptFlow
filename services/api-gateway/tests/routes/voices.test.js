'use strict';

const express = require('express');
const fs = require('fs');
const os = require('os');
const path = require('path');
const request = require('supertest');
const { voicesRouter } = require('../../src/routes/voices');

function buildApp(sharedDir) {
  const app = express();
  app.use(voicesRouter(sharedDir));
  return app;
}

function makeSharedDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'voices-'));
}

describe('routes/voices', () => {
  test('GET /v1/voices returns the catalog with a sample URL per voice', async () => {
    const sharedDir = makeSharedDir();
    const samplesDir = path.join(sharedDir, 'voice_samples');
    fs.mkdirSync(samplesDir, { recursive: true });
    fs.writeFileSync(
      path.join(samplesDir, 'catalog.json'),
      JSON.stringify([{ voice_id: 'vi_VN-vais1000-medium', language: 'vi', gender: 'female' }]),
    );

    const res = await request(buildApp(sharedDir)).get('/v1/voices');

    expect(res.status).toBe(200);
    expect(res.body).toHaveLength(1);
    expect(res.body[0].sample_audio_url).toBe('/v1/voices/vi_VN-vais1000-medium/sample');
  });

  test('GET /v1/voices reports 503 while the TTS Service has not written the catalog', async () => {
    const res = await request(buildApp(makeSharedDir())).get('/v1/voices');

    expect(res.status).toBe(503);
  });

  test('GET /v1/voices/:voiceId/sample serves the preview clip', async () => {
    const sharedDir = makeSharedDir();
    const samplesDir = path.join(sharedDir, 'voice_samples');
    fs.mkdirSync(samplesDir, { recursive: true });
    fs.writeFileSync(path.join(samplesDir, 'en_US-ryan-high.wav'), 'RIFF-fake-wav');

    const res = await request(buildApp(sharedDir)).get('/v1/voices/en_US-ryan-high/sample');

    expect(res.status).toBe(200);
    expect(res.headers['content-type']).toContain('audio/wav');
  });

  test('GET /v1/voices/:voiceId/sample does not escape the samples directory', async () => {
    const sharedDir = makeSharedDir();
    fs.mkdirSync(path.join(sharedDir, 'voice_samples'), { recursive: true });
    fs.writeFileSync(path.join(sharedDir, 'secret.wav'), 'private');

    const res = await request(buildApp(sharedDir)).get('/v1/voices/%2E%2E%2Fsecret/sample');

    expect(res.status).toBe(404);
  });
});
