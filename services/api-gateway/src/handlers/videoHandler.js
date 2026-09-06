'use strict';

const fs = require('fs');
const path = require('path');

/**
 * Streams a project's assembled video from the shared_artifacts volume.
 *
 * The orchestrator only stores the file's path on the shared volume
 * (`video_path`, e.g. "/shared/<project>/video/final.mp4") — it has no HTTP
 * route of its own to serve bytes back to a browser. This handler looks the
 * path up via the orchestrator client, then streams the file from the
 * Gateway's own mount of the same volume, supporting HTTP Range requests so
 * the `<video>` element can seek.
 *
 * @param {import('../clients/httpClient').HttpClient} orchestratorClient
 * @param {string} sharedDir - absolute path the shared_artifacts volume is mounted at, e.g. "/shared"
 */
function videoHandler(orchestratorClient, sharedDir) {
  return async (req, res) => {
    let projectRes;
    try {
      projectRes = await orchestratorClient.request({
        method: 'GET',
        path: `/v1/projects/${req.params.id}`,
      });
    } catch (err) {
      res.status(502).json({ error: 'upstream_unavailable', message: err.message });
      return;
    }

    if (projectRes.status !== 200) {
      res.status(projectRes.status).json(projectRes.body);
      return;
    }

    const videoPath = projectRes.body && projectRes.body.video_path;
    if (!videoPath) {
      res.status(404).json({ error: 'video_not_ready', message: 'Project has no video yet' });
      return;
    }

    const resolvedSharedDir = path.resolve(sharedDir);
    const resolvedVideoPath = path.resolve(videoPath);
    if (
      resolvedVideoPath !== resolvedSharedDir &&
      !resolvedVideoPath.startsWith(resolvedSharedDir + path.sep)
    ) {
      res.status(400).json({ error: 'invalid_video_path' });
      return;
    }

    fs.stat(resolvedVideoPath, (statErr, stat) => {
      if (statErr || !stat.isFile()) {
        res.status(404).json({ error: 'video_not_found' });
        return;
      }

      const range = req.headers.range;
      if (!range) {
        res.writeHead(200, {
          'Content-Type': 'video/mp4',
          'Content-Length': stat.size,
          'Accept-Ranges': 'bytes',
        });
        fs.createReadStream(resolvedVideoPath).pipe(res);
        return;
      }

      const [startStr, endStr] = range.replace(/bytes=/, '').split('-');
      const start = parseInt(startStr, 10);
      const end = endStr ? parseInt(endStr, 10) : stat.size - 1;

      res.writeHead(206, {
        'Content-Range': `bytes ${start}-${end}/${stat.size}`,
        'Accept-Ranges': 'bytes',
        'Content-Length': end - start + 1,
        'Content-Type': 'video/mp4',
      });
      fs.createReadStream(resolvedVideoPath, { start, end }).pipe(res);
    });
  };
}

module.exports = { videoHandler };
