'use strict';

const express = require('express');
const { proxyHandler } = require('../handlers/proxyHandler');

/**
 * Step 9 "Kế hoạch hình", forwarded straight to visual-service
 * (docs/contracts/visual-plan-v1.md).
 *
 * @param {import('../clients/httpClient').HttpClient} visualClient
 */
function visualPlanRouter(visualClient) {
  const router = express.Router();
  const base = '/v1/projects/:id/visual-plan';
  const plain = proxyHandler(visualClient, 'visual-service');
  router.get(base, plain);
  router.put(`${base}/units/:unit`, plain);
  router.put(`${base}/units/:unit/questions/:question`, plain);
  router.post(`${base}/units/:unit/split`, plain);
  router.post(`${base}/units/:unit/merge`, plain);
  router.post(`${base}/units/:unit/approve`, plain);
  router.post(`${base}/units/:unit/unapprove`, plain);
  router.post(`${base}/approve-all`, plain);
  router.post(`${base}/findings/:finding/apply`, plain);
  router.post(`${base}/findings/:finding/accept`, plain);
  router.post(`${base}/lock`, plain);
  router.get(`${base}/versions/:version`, plain);
  return router;
}

module.exports = { visualPlanRouter };
