'use strict';

const express = require('express');
const { proxyHandler } = require('../handlers/proxyHandler');

/**
 * Step 8 "Ngôn ngữ hình ảnh", forwarded straight to visual-service
 * (docs/contracts/visual-concept-v1.md). The routes that ask the AI go through
 * `visualAiClient`, which has no gateway timeout, like the AI routes of
 * authoring-service; the `X-External-AI` header passes through with the rest.
 *
 * @param {import('../clients/httpClient').HttpClient} visualClient
 * @param {import('../clients/httpClient').HttpClient} visualAiClient
 */
function visualConceptRouter(visualClient, visualAiClient) {
  const router = express.Router();
  const base = '/v1/projects/:id/visual-concept';
  const plain = proxyHandler(visualClient, 'visual-service');
  const ai = proxyHandler(visualAiClient, 'visual-service');
  router.get(base, plain);
  router.put(`${base}/refs`, plain);
  router.put(`${base}/model`, plain);
  router.post(`${base}/generate`, ai);
  router.post(`${base}/units/:key/generate`, ai);
  router.put(`${base}/units/:key`, plain);
  router.post(`${base}/units/:key/approve`, plain);
  router.post(`${base}/units/:key/unapprove`, plain);
  router.post(`${base}/approve-all`, plain);
  router.post(`${base}/review`, ai);
  router.post(`${base}/findings/:finding/apply`, plain);
  router.post(`${base}/findings/:finding/accept`, plain);
  router.post(`${base}/lock`, plain);
  router.get(`${base}/versions/:version`, plain);
  return router;
}

module.exports = { visualConceptRouter };
