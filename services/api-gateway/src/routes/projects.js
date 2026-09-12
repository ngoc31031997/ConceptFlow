'use strict';

const express = require('express');
const multer = require('multer');
const { proxyHandler } = require('../handlers/proxyHandler');
const { videoHandler } = require('../handlers/videoHandler');
const { clipHandler } = require('../handlers/clipHandler');
const { deleteProjectHandler } = require('../handlers/deleteProjectHandler');
const {
  thumbnailUploadHandler,
  thumbnailServeHandler,
  thumbnailInfoHandler,
} = require('../handlers/thumbnailUploadHandler');
const { musicUploadHandler, musicServeHandler, musicInfoHandler } = require('../handlers/musicUploadHandler');

const upload = multer({ storage: multer.memoryStorage(), limits: { fileSize: 2 * 1024 * 1024 } });
const musicUpload = multer({ storage: multer.memoryStorage(), limits: { fileSize: 20 * 1024 * 1024 } });

/**
 * `GET /v1/projects`, `GET /v1/projects/:id`, `GET /v1/voice-calibration` and
 * `POST /v1/projects/:id/retry` → Orchestrator Service.
 * `GET /v1/projects/:id/qc-report` → Orchestrator Service (CR-021 FR61.2).
 * `POST/GET /v1/projects/:id/clips` → Orchestrator Service (CR-007 D7).
 * `GET /v1/projects/:id/clips/:name/:preset` streams one generated clip from the shared volume.
 * `GET /v1/projects/:id/video` streams the assembled video from the shared volume.
 * `POST /v1/projects/:id/suggest-metadata` drafts SEO title/description/tags via Ollama —
 * proxied through `orchestratorAiClient` (a longer timeout than the default
 * 30s, since local LLM generation can take up to ~2 minutes).
 * `POST /v1/short-script-suggestions` drafts a standalone Shorts/TikTok
 * script via Ollama (CR-026 FR71) — same `orchestratorAiClient`, no :id.
 * `POST /v1/projects/:id/thumbnail` (multipart, field "thumbnail", max 2MB,
 * jpeg/png) saves a manually-uploaded thumbnail to the shared volume;
 * `GET /v1/projects/:id/thumbnail` serves it back for preview.
 * `POST /v1/projects/:id/music` (multipart, field "music", max 20MB,
 * mp3/wav/ogg/m4a) saves a manually-uploaded background-music file to the
 * shared volume, keyed by a client-generated project id (uploaded before the
 * render saga starts, same id later sent as `project_id`); `GET
 * /v1/projects/:id/music` serves it back for preview.
 * `DELETE /v1/projects/:id` removes the project's DB rows (Orchestrator) and its files (shared volume).
 * @param {import('../clients/httpClient').HttpClient} orchestratorClient
 * @param {string} sharedDir
 * @param {import('../clients/httpClient').HttpClient} orchestratorAiClient
 */
function projectsRouter(orchestratorClient, sharedDir, orchestratorAiClient) {
  const router = express.Router();
  router.get('/v1/projects', proxyHandler(orchestratorClient, 'orchestrator'));
  // CR-016 FR43.2 — tốc độ đọc đo được của từng giọng, để ước lượng thời lượng
  // lúc soạn khớp với giọng Creator thực sự dùng.
  router.get('/v1/voice-calibration', proxyHandler(orchestratorClient, 'orchestrator'));
  // CR-019 FR51.3/51.5 — hình dạng video: đọc danh sách, và lưu bản đã sửa
  // thành một phiên bản mới (không bao giờ ghi đè).
  router.get('/v1/formats', proxyHandler(orchestratorClient, 'orchestrator'));
  router.post('/v1/formats', proxyHandler(orchestratorClient, 'orchestrator'));
  router.get('/v1/projects/:id', proxyHandler(orchestratorClient, 'orchestrator'));
  router.get('/v1/projects/:id/video', videoHandler(orchestratorClient, sharedDir));
  router.post('/v1/projects/:id/retry', proxyHandler(orchestratorClient, 'orchestrator'));
  // CR-024 FR69.2/69.3 — hai lối ra khỏi cổng duyệt dàn ý.
  router.post('/v1/projects/:id/approve', proxyHandler(orchestratorClient, 'orchestrator'));
  router.post('/v1/projects/:id/reject', proxyHandler(orchestratorClient, 'orchestrator'));
  router.post('/v1/projects/:id/narration', proxyHandler(orchestratorClient, 'orchestrator'));
  router.get('/v1/projects/:id/qc-report', proxyHandler(orchestratorClient, 'orchestrator'));
  router.post('/v1/projects/:id/clips', proxyHandler(orchestratorClient, 'orchestrator'));
  router.get('/v1/projects/:id/clips', proxyHandler(orchestratorClient, 'orchestrator'));
  router.get('/v1/projects/:id/clips/:name/:preset', clipHandler(orchestratorClient, sharedDir));
  router.post(
    '/v1/projects/:id/suggest-metadata',
    proxyHandler(orchestratorAiClient || orchestratorClient, 'orchestrator'),
  );
  // CR-026 FR71 — same longer-timeout client as suggest-metadata: drafting a
  // whole script via the local model takes longer than a title/description.
  // No :id in the path (unlike suggest-metadata) — a Creator can draft a
  // short from a bare topic without an existing project.
  router.post(
    '/v1/short-script-suggestions',
    proxyHandler(orchestratorAiClient || orchestratorClient, 'orchestrator'),
  );
  router.post('/v1/projects/:id/thumbnail', upload.single('thumbnail'), thumbnailUploadHandler(sharedDir));
  router.get('/v1/projects/:id/thumbnail/info', thumbnailInfoHandler(sharedDir));
  router.get('/v1/projects/:id/thumbnail', thumbnailServeHandler(sharedDir));
  router.post('/v1/projects/:id/music', musicUpload.single('music'), musicUploadHandler(sharedDir));
  router.get('/v1/projects/:id/music/info', musicInfoHandler(sharedDir));
  router.get('/v1/projects/:id/music', musicServeHandler(sharedDir));
  router.delete('/v1/projects/:id', deleteProjectHandler(orchestratorClient, sharedDir));
  return router;
}

module.exports = { projectsRouter };
