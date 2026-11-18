'use strict';

const express = require('express');
const { proxyHandler } = require('../handlers/proxyHandler');

/**
 * Step 10 "Chuỗi hình", forwarded straight to visual-service
 * (docs/contracts/visual-sequence-v1.md). The routes that ask the AI go through
 * `visualAiClient`, which has no gateway timeout, like the AI routes of steps 8
 * and 9; the `X-External-AI` header passes through with the rest.
 *
 * @param {import('../clients/httpClient').HttpClient} visualClient
 * @param {import('../clients/httpClient').HttpClient} visualAiClient
 */
function visualSequenceRouter(visualClient, visualAiClient) {
  const router = express.Router();
  const base = '/v1/projects/:id/visual-sequence';
  const plain = proxyHandler(visualClient, 'visual-service');
  const ai = proxyHandler(visualAiClient, 'visual-service');
  router.get(base, plain);
  router.put(`${base}/model`, plain);
  router.post(`${base}/motifs/generate`, ai);
  router.post(`${base}/generate`, ai);
  router.post(`${base}/beats/:beat/generate`, ai);
  router.post(`${base}/units/:unit/generate`, ai);
  router.post(`${base}/review`, ai);
  router.put(`${base}/units/:unit`, plain);
  router.post(`${base}/units/:unit/approve`, plain);
  router.post(`${base}/units/:unit/unapprove`, plain);
  router.post(`${base}/approve-all`, plain);
  router.post(`${base}/beats/:beat/confirm`, plain);
  router.post(`${base}/motifs`, plain);
  router.post(`${base}/motifs/approve`, plain);
  router.post(`${base}/motifs/unapprove`, plain);
  router.put(`${base}/motifs/:motif`, plain);
  router.delete(`${base}/motifs/:motif`, plain);
  router.post(`${base}/findings/:finding/apply`, plain);
  router.post(`${base}/findings/:finding/accept`, plain);
  router.post(`${base}/lock`, plain);
  router.get(`${base}/versions/:version`, plain);
  return router;
}

module.exports = { visualSequenceRouter };
