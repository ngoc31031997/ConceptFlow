'use strict';

const fs = require('fs/promises');
const path = require('path');

/**
 * Deletes a project: forwards the delete to the Orchestrator (which removes
 * the `projects`/`saga_steps`/`outbox_events` rows in one transaction), then
 * — only once that succeeds — removes the project's artifact directory from
 * the shared volume, so a deleted video also stops taking up disk space.
 *
 * File cleanup happens here rather than in the Orchestrator because only the
 * Gateway has a (read-write) mount of shared_artifacts.
 *
 * @param {import('../clients/httpClient').HttpClient} orchestratorClient
 * @param {string} sharedDir - absolute path the shared_artifacts volume is mounted at, e.g. "/shared"
 */
function deleteProjectHandler(orchestratorClient, sharedDir) {
  return async (req, res) => {
    const projectId = req.params.id;

    let deleteRes;
    try {
      deleteRes = await orchestratorClient.request({
        method: 'DELETE',
        path: `/v1/projects/${projectId}`,
      });
    } catch (err) {
      res.status(502).json({ error: 'upstream_unavailable', message: err.message });
      return;
    }

    if (deleteRes.status !== 204) {
      res.status(deleteRes.status).json(deleteRes.body);
      return;
    }

    const resolvedSharedDir = path.resolve(sharedDir);
    const resolvedProjectDir = path.resolve(resolvedSharedDir, projectId);
    if (
      resolvedProjectDir !== resolvedSharedDir &&
      resolvedProjectDir.startsWith(resolvedSharedDir + path.sep)
    ) {
      await fs.rm(resolvedProjectDir, { recursive: true, force: true }).catch(() => {});
    }

    res.status(204).end();
  };
}

module.exports = { deleteProjectHandler };
