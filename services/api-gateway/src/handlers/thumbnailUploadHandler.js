'use strict';

const fs = require('fs');
const path = require('path');

const ALLOWED_MIME_TO_EXT = {
  'image/jpeg': '.jpg',
  'image/png': '.png',
};

/**
 * Saves an uploaded thumbnail image (multer memory buffer, field name
 * "thumbnail") to `<sharedDir>/<project_id>/thumbnail/thumbnail<ext>` — the
 * same shared_artifacts volume the rendering/assembly/publisher services
 * use, so the Publisher (which mounts it read-only) can read the file back
 * at publish time without another upload hop.
 *
 * @param {string} sharedDir
 */
function thumbnailUploadHandler(sharedDir) {
  return (req, res) => {
    if (!req.file) {
      res.status(400).json({ error: 'thumbnail file is required (field name "thumbnail")' });
      return;
    }

    const ext = ALLOWED_MIME_TO_EXT[req.file.mimetype];
    if (!ext) {
      res.status(400).json({ error: 'thumbnail must be image/jpeg or image/png' });
      return;
    }

    const projectId = req.params.id;
    const dir = path.join(path.resolve(sharedDir), projectId, 'thumbnail');
    const filePath = path.join(dir, `thumbnail${ext}`);

    fs.mkdir(dir, { recursive: true }, (mkdirErr) => {
      if (mkdirErr) {
        res.status(500).json({ error: 'failed to prepare storage for thumbnail' });
        return;
      }
      fs.writeFile(filePath, req.file.buffer, (writeErr) => {
        if (writeErr) {
          res.status(500).json({ error: 'failed to save thumbnail' });
          return;
        }
        res.status(200).json({ thumbnail_path: filePath });
      });
    });
  };
}

/**
 * Serves back the previously uploaded thumbnail (if any) so the frontend
 * can show a preview after a page reload, without persisting thumbnail
 * state anywhere else — existence on disk is the source of truth.
 *
 * @param {string} sharedDir
 */
function thumbnailServeHandler(sharedDir) {
  return (req, res) => {
    const projectId = req.params.id;
    const dir = path.join(path.resolve(sharedDir), projectId, 'thumbnail');

    const candidates = Object.values(ALLOWED_MIME_TO_EXT).map((ext) => path.join(dir, `thumbnail${ext}`));

    const tryNext = (index) => {
      if (index >= candidates.length) {
        res.status(404).json({ error: 'no thumbnail uploaded for this project' });
        return;
      }
      const candidate = candidates[index];
      fs.stat(candidate, (err, stat) => {
        if (err || !stat.isFile()) {
          tryNext(index + 1);
          return;
        }
        const contentType = candidate.endsWith('.png') ? 'image/png' : 'image/jpeg';
        res.writeHead(200, { 'Content-Type': contentType, 'Content-Length': stat.size });
        fs.createReadStream(candidate).pipe(res);
      });
    };
    tryNext(0);
  };
}

module.exports = { thumbnailUploadHandler, thumbnailServeHandler };
