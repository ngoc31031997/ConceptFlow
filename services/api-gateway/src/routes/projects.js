'use strict';

const express = require('express');
const multer = require('multer');
const { proxyHandler } = require('../handlers/proxyHandler');
const { videoHandler } = require('../handlers/videoHandler');
const { sharedFileHandler } = require('../handlers/sharedFileHandler');
const path = require('path');
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
 * `GET /v1/projects/:id/qc-report` → Orchestrator Service.
 * `GET /v1/projects/:id/video` streams the assembled video from the shared volume.
 * `POST /v1/projects/:id/suggest-metadata` drafts SEO title/description/tags via Ollama —
 * proxied through `orchestratorAiClient` (a longer timeout than the default
 * 30s, since local LLM generation can take up to ~2 minutes).
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
function projectsRouter(orchestratorClient, sharedDir, orchestratorAiClient, authoringClient, authoringAiClient) {
  const router = express.Router();
  // Authoring routes (prompts, the 1a/1b/1c artefacts, generate,
  // chain, progress, suggestions) belong to authoring-service. Falling back to
  // the orchestrator client keeps a gateway without AUTHORING_SERVICE_URL working
  // against the old single-service layout (and the route tests simple).
  const authoring = authoringClient || orchestratorClient;
  const authoringAi = authoringAiClient || authoringClient || orchestratorAiClient || orchestratorClient;
  router.get('/v1/projects', proxyHandler(orchestratorClient, 'orchestrator'));
  // The project row is created at wizard step 1
  // (topic entry), not at POST /v1/sagas/render, so authoring state written
  // from step 1 on always has a real project to hang off of.
  router.post('/v1/projects', proxyHandler(orchestratorClient, 'orchestrator'));
  router.patch('/v1/projects/:id/topic', proxyHandler(orchestratorClient, 'orchestrator'));
  // read-only history of every authoring field overwrite.
  router.get('/v1/projects/:id/authoring/history', proxyHandler(authoring, 'authoring-service'));
  // Append-only trace of failed runs (project_errors column).
  router.get('/v1/projects/:id/errors', proxyHandler(orchestratorClient, 'orchestrator'));
  // Journey log of the 13-step flow (project_events): one project, and the
  // cross-project feed the "Nhật ký" screen reads.
  router.get('/v1/projects/:id/events', proxyHandler(orchestratorClient, 'orchestrator'));
  router.get('/v1/events', proxyHandler(orchestratorClient, 'orchestrator'));
  // Tốc độ đọc đo được của từng giọng, để ước lượng thời lượng
  // lúc soạn khớp với giọng Creator thực sự dùng.
  router.get('/v1/voice-calibration', proxyHandler(orchestratorClient, 'orchestrator'));
  // Hình dạng video: đọc danh sách, và lưu bản đã sửa
  // thành một phiên bản mới (không bao giờ ghi đè).
  router.get('/v1/formats', proxyHandler(orchestratorClient, 'orchestrator'));
  router.post('/v1/formats', proxyHandler(orchestratorClient, 'orchestrator'));
  router.get('/v1/projects/:id', proxyHandler(orchestratorClient, 'orchestrator'));
  router.get('/v1/projects/:id/video', videoHandler(orchestratorClient, sharedDir));
  router.post('/v1/projects/:id/retry', proxyHandler(orchestratorClient, 'orchestrator'));
  // Stop the step a project is running (workers are told over a fanout), and
  // fork a project into a new one starting again at an earlier step.
  router.post('/v1/projects/:id/cancel', proxyHandler(orchestratorClient, 'orchestrator'));
  router.post('/v1/projects/:id/fork', proxyHandler(orchestratorClient, 'orchestrator'));
  // Hai lối ra khỏi cổng duyệt dàn ý.
  router.post('/v1/projects/:id/approve', proxyHandler(orchestratorClient, 'orchestrator'));
  router.post('/v1/projects/:id/reject', proxyHandler(orchestratorClient, 'orchestrator'));
  router.post('/v1/projects/:id/narration', proxyHandler(orchestratorClient, 'orchestrator'));
  router.get('/v1/projects/:id/qc-report', proxyHandler(orchestratorClient, 'orchestrator'));
  // Step 1 — saves the Story Architect output a Creator pasted back
  // after the external-AI round trip.
  router.post('/v1/projects/:id/authoring/story', proxyHandler(authoring, 'authoring-service'));
  // Step 2 — same shape, for the Visual Director's pasted storyboard.
  router.post('/v1/projects/:id/authoring/storyboard', proxyHandler(authoring, 'authoring-service'));
  // Step 3 — the Code step's pasted Remotion code.
  router.post('/v1/projects/:id/authoring/code', proxyHandler(authoring, 'authoring-service'));
  // Rehydrates every saved authoring output (story + storyboard +
  // code) so the wizard can restore state on reload/back-navigation.
  router.get('/v1/projects/:id/authoring', proxyHandler(authoring, 'authoring-service'));
  // Cách làm bước 1 (copy tay / gọi API), nhớ theo project nên
  // mở lại ở máy khác hay sau khi restart vẫn đúng chế độ đã chọn.
  router.put('/v1/projects/:id/authoring/mode', proxyHandler(authoring, 'authoring-service'));
  // Model Hive cho từng tab 1a/1b/1c — cùng kiểu với mode ở trên.
  router.put('/v1/projects/:id/authoring/models', proxyHandler(authoring, 'authoring-service'));
  // Wizard: settings = bước 2 (Cấu hình), PATCH từng field khi Creator đổi;
  // "Tiếp tục" gửi confirm để sang bước 3.
  router.patch('/v1/projects/:id/settings', proxyHandler(orchestratorClient, 'orchestrator'));
  // Records which wizard screen a draft was left on, so "Chi tiết" reopens there.
  router.post(
    '/v1/projects/:id/suggest-metadata',
    proxyHandler(authoringAi, 'authoring-service'),
  );
  // Idea and creative brief steps: each one's state and the AI writing it
  // (a model call, so the long-timeout client), plus the brief's target length and platform.
  router.get('/v1/projects/:id/idea', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/idea/generate', proxyHandler(authoringAi, 'authoring-service'));
  router.put('/v1/projects/:id/idea', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/idea/approve', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/idea/unapprove', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/idea/review', proxyHandler(authoringAi, 'authoring-service'));
  router.get('/v1/projects/:id/creative-brief', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/creative-brief/generate', proxyHandler(authoringAi, 'authoring-service'));
  router.put('/v1/projects/:id/creative-brief', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/creative-brief/approve', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/creative-brief/unapprove', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/creative-brief/review', proxyHandler(authoringAi, 'authoring-service'));
  router.get('/v1/projects/:id/brief', proxyHandler(authoring, 'authoring-service'));
  router.put('/v1/projects/:id/brief/target-duration', proxyHandler(authoring, 'authoring-service'));
  router.put('/v1/projects/:id/brief/platform', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/brief/generate', proxyHandler(authoringAi, 'authoring-service'));
  // Review units of every step (read, hand edit, redo with a note, approve,
  // impact). A redo calls the model, so it takes the long-timeout client.
  router.get('/v1/projects/:id/units/:kind/:key', proxyHandler(authoring, 'authoring-service'));
  router.put('/v1/projects/:id/units/:kind/:key', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/units/:kind/:key/redo', proxyHandler(authoringAi, 'authoring-service'));
  router.post('/v1/projects/:id/units/:kind/:key/approve', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/units/:kind/:key/unapprove', proxyHandler(authoring, 'authoring-service'));
  router.get('/v1/projects/:id/units/:kind/:key/impact', proxyHandler(authoring, 'authoring-service'));
  // Nghiên cứu: câu hỏi cần tra, prompt chép ra công cụ ngoài, dán kết quả và
  // bảng dữ kiện. Chỉ việc AI sinh câu hỏi gọi model nên dùng client timeout dài;
  // tải trang kiểm link chạy nền trong authoring-service.
  router.get('/v1/projects/:id/research', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/research/questions/generate', proxyHandler(authoringAi, 'authoring-service'));
  router.post('/v1/projects/:id/research/questions', proxyHandler(authoring, 'authoring-service'));
  router.delete('/v1/projects/:id/research/questions/:qid', proxyHandler(authoring, 'authoring-service'));
  router.get('/v1/projects/:id/research/prompt', proxyHandler(authoring, 'authoring-service'));
  router.get('/v1/projects/:id/research/follow-up-prompt', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/research/paste', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/research/facts', proxyHandler(authoring, 'authoring-service'));
  router.put('/v1/projects/:id/research/facts/:fid', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/research/verify', proxyHandler(authoring, 'authoring-service'));
  router.put('/v1/projects/:id/research/skip', proxyHandler(authoring, 'authoring-service'));
  // Câu chuyện: một dòng project_story (câu hỏi trung tâm, câu trả lời cốt lõi, mạch kể).
  // Chỉ lượt AI viết gọi model nên dùng client timeout dài.
  router.get('/v1/projects/:id/story', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/story/generate', proxyHandler(authoringAi, 'authoring-service'));
  router.put('/v1/projects/:id/story', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/story/approve', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/story/unapprove', proxyHandler(authoring, 'authoring-service'));
  // Bước Beat sheet: bộ khung, cả bảng, thứ tự, các lượt AI và từng beat (sửa, làm lại, duyệt, tác động).
  router.get('/v1/projects/:id/beats', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/beats/skeleton', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/beats/skeleton/approve', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/beats/generate', proxyHandler(authoringAi, 'authoring-service'));
  router.post('/v1/projects/:id/beats', proxyHandler(authoringAi, 'authoring-service'));
  router.put('/v1/projects/:id/beats/order', proxyHandler(authoring, 'authoring-service'));
  router.delete('/v1/projects/:id/beats/:key', proxyHandler(authoring, 'authoring-service'));
  router.put('/v1/projects/:id/beats/:key', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/beats/:key/redo', proxyHandler(authoringAi, 'authoring-service'));
  router.post('/v1/projects/:id/beats/:key/approve', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/beats/:key/unapprove', proxyHandler(authoring, 'authoring-service'));
  router.get('/v1/projects/:id/beats/:key/impact', proxyHandler(authoring, 'authoring-service'));
  // Bước Viết kịch bản: lời thoại từng beat đi qua /units/script_beat/*; ở đây là
  // cả màn, các lượt AI (viết, gợi ý) và đưa câu đi đọc. Đọc giọng chạy nền qua
  // orchestrator và tts nên lời gọi chỉ xếp câu vào hàng chờ.
  router.get('/v1/projects/:id/script', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/script/generate', proxyHandler(authoringAi, 'authoring-service'));
  router.post('/v1/projects/:id/script/suggest', proxyHandler(authoringAi, 'authoring-service'));
  router.post('/v1/projects/:id/script/speak', proxyHandler(authoring, 'authoring-service'));
  // Bước Rà soát và chốt: ghi chú bàn giao từng beat đi qua /units/handoff_beat/*; ở đây
  // là cả màn, lượt AI rà soát và viết ghi chú bàn giao, xử lý từng mục, thuật ngữ và chốt.
  router.get('/v1/projects/:id/script-lock', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/script-lock/review', proxyHandler(authoringAi, 'authoring-service'));
  router.post('/v1/projects/:id/script-lock/findings/:finding/apply', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/script-lock/findings/:finding/handoff', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/script-lock/findings/:finding/accept', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/script-lock/notes', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/script-lock/glossary', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/script-lock/handoff/generate', proxyHandler(authoringAi, 'authoring-service'));
  router.post('/v1/projects/:id/script-lock/handoff/:key/confirm', proxyHandler(authoring, 'authoring-service'));
  router.post('/v1/projects/:id/script-lock/lock', proxyHandler(authoring, 'authoring-service'));
  // File giọng của từng câu, do tts ghi vào /shared/{project}/audio. Tên file gồm
  // mã câu và khoá nội dung nên không bao giờ đổi; chỉ tên đúng mẫu mới được phát.
  router.get('/v1/projects/:id/audio/:file', (req, res, next) => {
    if (!/^[A-Za-z0-9_-]{1,64}$/.test(req.params.id)) {
      res.status(404).json({ error: 'file_not_found' });
      return;
    }
    sharedFileHandler(path.join(sharedDir, req.params.id, 'audio'), {
      param: 'file',
      namePattern: /^b[1-9][0-9]*-[1-9][0-9]*_[0-9a-f]{12}\.wav$/,
      contentType: 'audio/wav',
    })(req, res, next);
  });
  router.post('/v1/projects/:id/thumbnail', upload.single('thumbnail'), thumbnailUploadHandler(sharedDir));
  router.get('/v1/projects/:id/thumbnail/info', thumbnailInfoHandler(sharedDir));
  router.get('/v1/projects/:id/thumbnail', thumbnailServeHandler(sharedDir));
  router.post('/v1/projects/:id/music', musicUpload.single('music'), musicUploadHandler(sharedDir));
  router.get('/v1/projects/:id/music/info', musicInfoHandler(sharedDir));
  router.get('/v1/projects/:id/music', musicServeHandler(sharedDir));
  // One progress endpoint for every long call. The delete saga
  // (`delete:<project_id>`) is the orchestrator's; everything else is a
  // suggestion running in authoring-service.
  const toAuthoring = proxyHandler(authoring, 'authoring-service');
  const toOrchestrator = proxyHandler(orchestratorClient, 'orchestrator');
  router.get('/v1/operations/:operationId', (req, res, next) =>
    (req.params.operationId.startsWith('delete:') ? toOrchestrator : toAuthoring)(req, res, next),
  );
  // Stopping a model call, and finding the ones a project still has running,
  // are authoring-service's too.
  router.post('/v1/operations/:operationId/cancel', toAuthoring);
  router.get('/v1/projects/:id/operations', toAuthoring);
  router.delete('/v1/projects/:id', deleteProjectHandler(orchestratorClient, sharedDir));
  return router;
}

module.exports = { projectsRouter };
