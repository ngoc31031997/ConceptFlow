'use strict';

const fs = require('fs');
const path = require('path');

const ALLOWED_MIME_TO_EXT = {
  'audio/mpeg': '.mp3',
  'audio/wav': '.wav',
  'audio/x-wav': '.wav',
  'audio/ogg': '.ogg',
  'audio/mp4': '.m4a',
  'audio/x-m4a': '.m4a',
};

/**
 * Saves an uploaded background-music file (multer memory buffer, field name
 * "music") to `<sharedDir>/<project_id>/music/background<ext>` — the same
 * shared_artifacts volume Video Assembly mounts, so `background_music_path`
 * sent with the render saga resolves to a real file without another upload
 * hop. Mirrors thumbnailUploadHandler.js's shape.
 *
 * @param {string} sharedDir
 */
function musicUploadHandler(sharedDir) {
  return (req, res) => {
    if (!req.file) {
      res.status(400).json({ error: 'music file is required (field name "music")' });
      return;
    }

    const ext = ALLOWED_MIME_TO_EXT[req.file.mimetype];
    if (!ext) {
      res.status(400).json({ error: 'music must be audio/mpeg, audio/wav, audio/ogg, or audio/mp4' });
      return;
    }

    const projectId = req.params.id;
    const dir = path.join(path.resolve(sharedDir), projectId, 'music');
    const filePath = path.join(dir, `background${ext}`);

    fs.mkdir(dir, { recursive: true }, (mkdirErr) => {
      if (mkdirErr) {
        res.status(500).json({ error: 'failed to prepare storage for music' });
        return;
      }
      fs.writeFile(filePath, req.file.buffer, (writeErr) => {
        if (writeErr) {
          res.status(500).json({ error: 'failed to save music' });
          return;
        }
        res.status(200).json({ background_music_path: filePath });
      });
    });
  };
}

/**
 * Serves back the previously uploaded background music (if any) so the
 * frontend can offer a preview player, without persisting music state
 * anywhere else — existence on disk is the source of truth.
 *
 * @param {string} sharedDir
 */
function musicServeHandler(sharedDir) {
  return (req, res) => {
    const projectId = req.params.id;
    const dir = path.join(path.resolve(sharedDir), projectId, 'music');

    const candidates = [...new Set(Object.values(ALLOWED_MIME_TO_EXT))].map((ext) =>
      path.join(dir, `background${ext}`),
    );

    const tryNext = (index) => {
      if (index >= candidates.length) {
        res.status(404).json({ error: 'no music uploaded for this project' });
        return;
      }
      const candidate = candidates[index];
      fs.stat(candidate, (err, stat) => {
        if (err || !stat.isFile()) {
          tryNext(index + 1);
          return;
        }
        const contentType =
          Object.entries(ALLOWED_MIME_TO_EXT).find(([, e]) => candidate.endsWith(e))?.[0] ??
          'application/octet-stream';
        res.writeHead(200, { 'Content-Type': contentType, 'Content-Length': stat.size });
        fs.createReadStream(candidate).pipe(res);
      });
    };
    tryNext(0);
  };
}

/**
 * Reports whether background music already exists for this project and its
 * absolute path (needed by the frontend to restore `background_music_path`
 * after a page reload, when it only has the project id).
 *
 * @param {string} sharedDir
 */
function musicInfoHandler(sharedDir) {
  return (req, res) => {
    const projectId = req.params.id;
    const dir = path.join(path.resolve(sharedDir), projectId, 'music');
    const candidates = [...new Set(Object.values(ALLOWED_MIME_TO_EXT))].map((ext) =>
      path.join(dir, `background${ext}`),
    );

    const tryNext = (index) => {
      if (index >= candidates.length) {
        res.status(200).json({ exists: false, background_music_path: null });
        return;
      }
      fs.stat(candidates[index], (err, stat) => {
        if (err || !stat.isFile()) {
          tryNext(index + 1);
          return;
        }
        res.status(200).json({ exists: true, background_music_path: candidates[index] });
      });
    };
    tryNext(0);
  };
}

module.exports = { musicUploadHandler, musicServeHandler, musicInfoHandler };
