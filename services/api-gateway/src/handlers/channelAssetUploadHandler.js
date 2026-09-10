'use strict';

const crypto = require('crypto');
const fsp = require('fs/promises');
const os = require('os');
const path = require('path');

const { probeVideo: defaultProbeVideo } = require('../clients/mediaProbe');
const { UpstreamUnavailableError } = require('../clients/httpClient');

const ALLOWED_VIDEO_MIME_TO_EXT = {
  'video/mp4': '.mp4',
  'video/quicktime': '.mov',
  'video/webm': '.webm',
};

const ALLOWED_AUDIO_MIME_TO_EXT = {
  'audio/mpeg': '.mp3',
  'audio/wav': '.wav',
  'audio/x-wav': '.wav',
  'audio/ogg': '.ogg',
  'audio/mp4': '.m4a',
  'audio/x-m4a': '.m4a',
};

const DEFAULT_RENDER_QUALITY = '1080p60';

/** `asset_role` values on the normalize command (CR-023 FR65.4/FR66.5). */
const ASSET_ROLE_VIDEO = 'video';
const ASSET_ROLE_MUSIC = 'music';

/** FR66.7 — an intro/outro sting is a 16:9 clip of at most 5 seconds. */
const MAX_DURATION_SECONDS = 5;
const ASPECT_RATIO = 16 / 9;
// Tolerances: a hair of slack for encoders that report e.g. 1918x1080, and
// for containers whose duration rounds up a few milliseconds past 5.000.
const ASPECT_TOLERANCE = 0.02;
const DURATION_TOLERANCE_SECONDS = 0.05;

/**
 * Intro/outro assets belong to the CHANNEL, not to any single project, so
 * unlike thumbnails and background music they are stored outside the
 * per-project subtree: `<sharedDir>/channel-assets/<kind>/<name><ext>`.
 */
function channelAssetPath(sharedDir, kind, basename, ext) {
  return path.join(path.resolve(sharedDir), 'channel-assets', kind, `${basename}${ext}`);
}

function sha256(buffer) {
  return crypto.createHash('sha256').update(buffer).digest('hex');
}

function readRenderQuality(req) {
  const fromQuery = req.query && req.query.render_quality;
  const fromBody = req.body && req.body.render_quality;
  return fromQuery || fromBody || DEFAULT_RENDER_QUALITY;
}

/**
 * Asks Orchestrator to queue the `normalize_channel_asset` AMQP command for a
 * file this Gateway has just written to the shared volume (CR-023 correction:
 * there is no HTTP server on video-assembly — Orchestrator is the only
 * service with an HTTP layer, and it publishes the command).
 */
async function triggerNormalize(orchestratorClient, req, kind, filePath, sourceHash, renderQuality, assetRole) {
  return orchestratorClient.request({
    method: 'POST',
    path: `/v1/channel-assets/${kind}`,
    headers: { 'content-type': 'application/json', 'x-request-id': req.requestId },
    body: {
      file_path: filePath,
      source_hash: sourceHash,
      render_quality: renderQuality,
      // `asset_role` is what tells video-assembly whether file_path is the
      // sting clip itself or the music bed it must mix into it (FR66.5) —
      // the two uploads otherwise look identical downstream.
      asset_role: assetRole,
    },
  });
}

/**
 * Writes `buffer` to `filePath`, then triggers the normalize command and
 * mirrors Orchestrator's status back to the Creator.
 */
async function saveAndTrigger(
  res,
  req,
  orchestratorClient,
  kind,
  filePath,
  buffer,
  renderQuality,
  assetRole,
  extraBody
) {
  await fsp.mkdir(path.dirname(filePath), { recursive: true });
  await fsp.writeFile(filePath, buffer);

  const sourceHash = sha256(buffer);
  let upstream;
  try {
    upstream = await triggerNormalize(
      orchestratorClient,
      req,
      kind,
      filePath,
      sourceHash,
      renderQuality,
      assetRole
    );
  } catch (err) {
    if (err instanceof UpstreamUnavailableError) {
      res.status(502).json({ error: 'upstream_unavailable', service: 'orchestrator' });
      return;
    }
    throw err;
  }

  if (upstream.status >= 400) {
    res.status(upstream.status).send(upstream.body);
    return;
  }

  res.status(202).json({
    kind,
    status: 'queued',
    file_path: filePath,
    source_hash: sourceHash,
    render_quality: renderQuality,
    ...extraBody,
  });
}

/**
 * `POST /v1/channel-assets/:kind` (kind = intro | outro), multipart field
 * "video". Validates the clip at upload time (FR66.7) so the Creator gets a
 * concrete reason immediately instead of a silent no-op an AMQP hop later —
 * a rejected file is never written to the shared volume (it is probed from a
 * throwaway file under the OS temp dir, which is removed either way).
 *
 * @param {string} sharedDir
 * @param {import('../clients/httpClient').HttpClient} orchestratorClient
 * @param {{ probeVideo?: typeof defaultProbeVideo }} [deps] - `probeVideo` is
 *   injectable so tests need no real ffprobe binary.
 */
function channelAssetVideoUploadHandler(sharedDir, orchestratorClient, deps = {}) {
  const probeVideo = deps.probeVideo || defaultProbeVideo;

  return async (req, res) => {
    const kind = req.params.kind;
    if (kind !== 'intro' && kind !== 'outro') {
      res.status(400).json({ error: "kind must be 'intro' or 'outro'" });
      return;
    }
    if (!req.file) {
      res.status(400).json({ error: 'video file is required (field name "video")' });
      return;
    }
    const ext = ALLOWED_VIDEO_MIME_TO_EXT[req.file.mimetype];
    if (!ext) {
      res.status(400).json({
        error: `unsupported video type "${req.file.mimetype}" — must be video/mp4, video/quicktime, or video/webm`,
      });
      return;
    }

    const tempPath = path.join(os.tmpdir(), `channel-asset-${crypto.randomUUID()}${ext}`);
    let probed;
    try {
      await fsp.writeFile(tempPath, req.file.buffer);
      probed = await probeVideo(tempPath);
    } catch (err) {
      await fsp.rm(tempPath, { force: true }).catch(() => {});
      res.status(400).json({ error: `could not read the uploaded video: ${err.message}` });
      return;
    }
    await fsp.rm(tempPath, { force: true }).catch(() => {});

    const ratio = probed.width / probed.height;
    if (Math.abs(ratio - ASPECT_RATIO) > ASPECT_TOLERANCE) {
      res.status(400).json({
        error: `aspect ratio must be 16:9, got ${probed.width}x${probed.height} (${ratio.toFixed(3)}:1)`,
      });
      return;
    }
    if (probed.durationSeconds > MAX_DURATION_SECONDS + DURATION_TOLERANCE_SECONDS) {
      res.status(400).json({
        error: `duration must be at most ${MAX_DURATION_SECONDS} seconds, got ${probed.durationSeconds.toFixed(2)}s`,
      });
      return;
    }

    const filePath = channelAssetPath(sharedDir, kind, 'source', ext);
    try {
      await saveAndTrigger(
        res,
        req,
        orchestratorClient,
        kind,
        filePath,
        req.file.buffer,
        readRenderQuality(req),
        ASSET_ROLE_VIDEO,
        { duration_seconds: probed.durationSeconds }
      );
    } catch (err) {
      res.status(500).json({ error: `failed to save ${kind} video: ${err.message}` });
    }
  };
}

/**
 * `POST /v1/channel-assets/:kind-music`, multipart field "music" — the sting's
 * audio bed, which video-assembly normalizes to -14 LUFS (FR66.5); the Gateway
 * only stores the file and triggers the same normalize command.
 *
 * @param {string} sharedDir
 * @param {import('../clients/httpClient').HttpClient} orchestratorClient
 */
function channelAssetMusicUploadHandler(sharedDir, orchestratorClient) {
  return async (req, res) => {
    const kind = req.params.kind;
    if (kind !== 'intro' && kind !== 'outro') {
      res.status(400).json({ error: "kind must be 'intro' or 'outro'" });
      return;
    }
    if (!req.file) {
      res.status(400).json({ error: 'music file is required (field name "music")' });
      return;
    }
    const ext = ALLOWED_AUDIO_MIME_TO_EXT[req.file.mimetype];
    if (!ext) {
      res.status(400).json({
        error: `unsupported audio type "${req.file.mimetype}" — must be audio/mpeg, audio/wav, audio/ogg, or audio/mp4`,
      });
      return;
    }

    const filePath = channelAssetPath(sharedDir, kind, 'music', ext);
    try {
      await saveAndTrigger(
        res,
        req,
        orchestratorClient,
        kind,
        filePath,
        req.file.buffer,
        readRenderQuality(req),
        ASSET_ROLE_MUSIC,
        { music_path: filePath }
      );
    } catch (err) {
      res.status(500).json({ error: `failed to save ${kind} music: ${err.message}` });
    }
  };
}

module.exports = {
  channelAssetVideoUploadHandler,
  channelAssetMusicUploadHandler,
  ALLOWED_VIDEO_MIME_TO_EXT,
  ALLOWED_AUDIO_MIME_TO_EXT,
};
