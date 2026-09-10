'use strict';

const express = require('express');
const multer = require('multer');
const { proxyHandler } = require('../handlers/proxyHandler');
const {
  channelAssetVideoUploadHandler,
  channelAssetMusicUploadHandler,
} = require('../handlers/channelAssetUploadHandler');

// A ≤5s 16:9 sting is small even at upload quality; the cap is a guard rail,
// not a budget. Music beds are capped like project background music (20MB).
const videoUpload = multer({ storage: multer.memoryStorage(), limits: { fileSize: 50 * 1024 * 1024 } });
const musicUpload = multer({ storage: multer.memoryStorage(), limits: { fileSize: 20 * 1024 * 1024 } });

/**
 * CR-023 — channel-wide intro/outro stings (FR65.1, FR66.5, FR66.7, FR67.4).
 *
 * These assets belong to the channel, not to a project, so the routes carry no
 * `:id`: `POST /v1/channel-assets/intro|outro` (multipart field "video") and
 * `POST /v1/channel-assets/intro-music|outro-music` (field "music") write the
 * file to `<sharedDir>/channel-assets/<kind>/` and then ask Orchestrator to
 * publish the `normalize_channel_asset` AMQP command video-assembly consumes.
 * `GET /v1/channel-assets/preview` is a plain proxy of Orchestrator's own
 * `channel_asset_pointers` projection.
 *
 * @param {import('../clients/httpClient').HttpClient} orchestratorClient
 * @param {string} sharedDir
 * @param {{ probeVideo?: Function }} [deps] - injected in tests so no real
 *   ffprobe binary is needed.
 */
function channelAssetsRouter(orchestratorClient, sharedDir, deps = {}) {
  const router = express.Router();

  // Registered before the video route so "intro-music" is never matched as a
  // kind by the more general pattern.
  router.post(
    '/v1/channel-assets/:kind(intro|outro)-music',
    musicUpload.single('music'),
    channelAssetMusicUploadHandler(sharedDir, orchestratorClient),
  );
  router.post(
    '/v1/channel-assets/:kind(intro|outro)',
    videoUpload.single('video'),
    channelAssetVideoUploadHandler(sharedDir, orchestratorClient, deps),
  );
  router.get('/v1/channel-assets/preview', proxyHandler(orchestratorClient, 'orchestrator'));

  return router;
}

module.exports = { channelAssetsRouter };
