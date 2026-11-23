'use strict';

const express = require('express');
const { proxyHandler } = require('../handlers/proxyHandler');

/**
 * Step 11 "Storyboard", forwarded straight to visual-service
 * (docs/contracts/storyboard-v1.md). Every route that asks the AI, draws an
 * asset or draws a frame goes through `visualAiClient`, which has no gateway
 * timeout, like the AI routes of steps 8 to 10; the `X-External-AI` header
 * passes through with the rest. Frames and asset previews come back as
 * `image/png`, which the proxy forwards with its type and cache headers.
 *
 * @param {import('../clients/httpClient').HttpClient} visualClient
 * @param {import('../clients/httpClient').HttpClient} visualAiClient
 */
function visualStoryboardRouter(visualClient, visualAiClient) {
  const router = express.Router();
  const base = '/v1/projects/:id/storyboard';
  const plain = proxyHandler(visualClient, 'visual-service');
  const ai = proxyHandler(visualAiClient, 'visual-service');
  router.get(base, plain);
  router.put(`${base}/model`, plain);

  // Assets.
  router.post(`${base}/assets/generate`, ai);
  router.post(`${base}/assets/detail`, ai);
  router.post(`${base}/assets/:asset/detail`, ai);
  router.post(`${base}/assets/approve`, plain);
  router.post(`${base}/assets/unapprove`, plain);
  router.post(`${base}/assets`, plain);
  router.put(`${base}/assets/:asset`, plain);
  router.delete(`${base}/assets/:asset`, plain);
  router.get(`${base}/assets/:asset/preview`, plain);
  router.post(`${base}/assets/:asset/draw`, ai);
  router.post(`${base}/assets/:asset/redraw`, ai);
  router.post(`${base}/assets/:asset/approve`, plain);
  router.post(`${base}/assets/:asset/unapprove`, plain);

  // Shots and the frames drawn from them.
  router.post(`${base}/generate`, ai);
  router.post(`${base}/units/:unit/generate`, ai);
  router.post(`${base}/units/:unit/shots`, ai);
  router.put(`${base}/units/:unit/shots/:shot`, ai);
  router.delete(`${base}/units/:unit/shots/:shot`, plain);
  router.post(`${base}/units/:unit/shots/:shot/frame`, ai);
  router.post(`${base}/units/:unit/shots/:shot/approve`, plain);
  router.post(`${base}/units/:unit/shots/:shot/unapprove`, plain);
  router.post(`${base}/approve-all`, plain);
  router.get(`${base}/frames/:shot`, plain);

  // Review and lock.
  router.post(`${base}/review`, ai);
  router.post(`${base}/findings/:finding/apply`, plain);
  router.post(`${base}/findings/:finding/accept`, plain);
  router.post(`${base}/lock`, plain);
  router.get(`${base}/versions/:version`, plain);

  // AI runs with a preview: start builds the prompt (and, on the API channel,
  // asks the model in the background inside visual-service); reply reads a
  // pasted or extension reply; apply stores it and redraws frames.
  router.post(`${base}/ai-runs`, ai);
  router.get(`${base}/ai-runs`, plain);
  router.get(`${base}/ai-runs/:run`, plain);
  router.post(`${base}/ai-runs/:run/reply`, ai);
  router.post(`${base}/ai-runs/:run/apply`, ai);
  router.post(`${base}/ai-runs/:run/cancel`, plain);
  return router;
}

module.exports = { visualStoryboardRouter };
