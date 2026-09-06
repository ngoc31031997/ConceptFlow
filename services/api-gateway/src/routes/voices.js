'use strict';

const express = require('express');
const fs = require('fs');
const path = require('path');

const VOICE_SAMPLES_DIR = 'voice_samples';
const CATALOG_FILE = 'catalog.json';

/**
 * `GET /v1/voices` and `GET /v1/voices/:voiceId/sample` (CR-001).
 *
 * The catalog and its preview clips are written to the shared volume by the
 * TTS Service at startup rather than fetched over HTTP: the TTS Service is a
 * pure AMQP consumer (ADR-0014 removed its REST surface) and reintroducing a
 * web server there just to list four static voices is not worth the
 * dependency. The Gateway already serves other shared-volume artifacts
 * (project video, thumbnail) the same way.
 *
 * @param {string} sharedDir
 */
function voicesRouter(sharedDir) {
  const router = express.Router();
  const samplesDir = path.join(path.resolve(sharedDir), VOICE_SAMPLES_DIR);

  router.get('/v1/voices', (req, res) => {
    fs.readFile(path.join(samplesDir, CATALOG_FILE), 'utf8', (err, contents) => {
      if (err) {
        // The TTS Service has not finished its first startup yet.
        res.status(503).json({ error: 'voice catalog is not available yet' });
        return;
      }
      let catalog;
      try {
        catalog = JSON.parse(contents);
      } catch {
        res.status(500).json({ error: 'voice catalog is corrupt' });
        return;
      }
      res.status(200).json(
        catalog.map((voice) => ({
          ...voice,
          sample_audio_url: `/v1/voices/${encodeURIComponent(voice.voice_id)}/sample`,
        })),
      );
    });
  });

  router.get('/v1/voices/:voiceId/sample', (req, res) => {
    // path.basename strips any traversal attempt before it reaches the filesystem.
    const fileName = `${path.basename(req.params.voiceId)}.wav`;
    const filePath = path.join(samplesDir, fileName);

    fs.access(filePath, fs.constants.R_OK, (err) => {
      if (err) {
        res.status(404).json({ error: 'voice sample not found' });
        return;
      }
      res.type('audio/wav').sendFile(filePath);
    });
  });

  return router;
}

module.exports = { voicesRouter };
