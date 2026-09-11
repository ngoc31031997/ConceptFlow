'use strict';

const fs = require('fs');
const path = require('path');

/**
 * Streams one generated vertical clip from the shared_artifacts volume
 * (CR-007 D7/FR20.1).
 *
 * Mirrors videoHandler's shape: Orchestrator only stores the clip's
 * output_path (video-assembly wrote it there directly, per
 * cr-007-low-level-design.md D5), so this handler asks Orchestrator for the
 * project's clip list, finds the one matching :name/:preset, then streams the
 * file from the Gateway's own mount of the same volume. This is deliberately
 * NOT re-slugified here — the LLD flagged two independent slugify
 * implementations (Python in video-assembly, JS here) as a real drift risk,
 * so the path comes from Orchestrator's stored output_path instead of being
 * recomputed.
 *
 * @param {import('../clients/httpClient').HttpClient} orchestratorClient
 * @param {string} sharedDir - absolute path the shared_artifacts volume is mounted at, e.g. "/shared"
 */
function clipHandler(orchestratorClient, sharedDir) {
  return async (req, res) => {
    let clipsRes;
    try {
      clipsRes = await orchestratorClient.request({
        method: 'GET',
        path: `/v1/projects/${req.params.id}/clips`,
      });
    } catch (err) {
      res.status(502).json({ error: 'upstream_unavailable', message: err.message });
      return;
    }

    if (clipsRes.status !== 200) {
      res.status(clipsRes.status).json(clipsRes.body);
      return;
    }

    const clips = (clipsRes.body && clipsRes.body.clips) || [];
    const clip = clips.find(
      (c) => c.name === req.params.name && c.preset === req.params.preset,
    );
    if (!clip || clip.status !== 'ok' || !clip.output_path) {
      res.status(404).json({ error: 'clip_not_found' });
      return;
    }

    const resolvedSharedDir = path.resolve(sharedDir);
    const resolvedClipPath = path.resolve(clip.output_path);
    if (
      resolvedClipPath !== resolvedSharedDir &&
      !resolvedClipPath.startsWith(resolvedSharedDir + path.sep)
    ) {
      res.status(400).json({ error: 'invalid_clip_path' });
      return;
    }

    fs.stat(resolvedClipPath, (statErr, stat) => {
      if (statErr || !stat.isFile()) {
        res.status(404).json({ error: 'clip_not_found' });
        return;
      }

      const range = req.headers.range;
      if (!range) {
        res.writeHead(200, {
          'Content-Type': 'video/mp4',
          'Content-Length': stat.size,
          'Accept-Ranges': 'bytes',
        });
        fs.createReadStream(resolvedClipPath).pipe(res);
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
      fs.createReadStream(resolvedClipPath, { start, end }).pipe(res);
    });
  };
}

module.exports = { clipHandler };
