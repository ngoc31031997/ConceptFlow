'use strict';

const fs = require('fs/promises');
const path = require('path');

/** Directories under `<sharedDir>/<project_id>/` that the Gateway itself writes
 * (uploads). Everything else there belongs to tts, rendering or video-assembly
 * and is removed by their own purge_project_artifacts step. */
const GATEWAY_OWNED_DIRS = ['thumbnail', 'music'];

/**
 * Deletes a project (CR-040 FR114.2). The Orchestrator runs the delete saga:
 * it refuses (409) while a step is running, marks the project `deleting`, and
 * has tts/rendering/video-assembly remove their own files before the row goes.
 * Answered 202 — the project disappears from the list at once and is gone for
 * good when every service has confirmed.
 *
 * The Gateway removes only what it wrote itself (thumbnail and music uploads),
 * never the whole project directory: a recursive delete here could race a
 * service that is still writing.
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

    if (deleteRes.status !== 202 && deleteRes.status !== 204) {
      res.status(deleteRes.status).json(deleteRes.body);
      return;
    }

    const resolvedSharedDir = path.resolve(sharedDir);
    const resolvedProjectDir = path.resolve(resolvedSharedDir, projectId);
    if (
      resolvedProjectDir !== resolvedSharedDir &&
      resolvedProjectDir.startsWith(resolvedSharedDir + path.sep)
    ) {
      for (const name of GATEWAY_OWNED_DIRS) {
        await fs.rm(path.join(resolvedProjectDir, name), { recursive: true, force: true }).catch(() => {});
      }
    }

    if (deleteRes.status === 202) {
      res.status(202).json(deleteRes.body);
      return;
    }
    res.status(204).end();
  };
}

module.exports = { deleteProjectHandler };
