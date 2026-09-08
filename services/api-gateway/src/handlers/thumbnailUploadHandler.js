'use strict';

const fs = require('fs');
const path = require('path');

const ALLOWED_MIME_TO_EXT = {
  'image/jpeg': '.jpg',
  'image/png': '.png',
};

/**
 * Thumbnails to look for, most-preferred first.
 *
 * A Creator's own upload always wins over `auto.jpg`, the candidate Video
 * Assembly extracts from the finished video (CR-006 FR16). The auto one exists
 * so a project is never publishable without any thumbnail at all — YouTube
 * otherwise picks a frame itself, and its choice is rarely a good one.
 */
const THUMBNAIL_FILENAMES = ['thumbnail.jpg', 'thumbnail.png', 'auto.jpg'];

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

    const candidates = THUMBNAIL_FILENAMES.map((name) => path.join(dir, name));

    const tryNext = (index) => {
      if (index >= candidates.length) {
        res.status(404).json({ error: 'no thumbnail available for this project' });
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

/**
 * Reports whether a thumbnail already exists for this project and its
 * absolute path (needed by the frontend to include `thumbnail_path` in the
 * publish payload after a page reload, when it only has the project id).
 *
 * @param {string} sharedDir
 */
function thumbnailInfoHandler(sharedDir) {
  return (req, res) => {
    const projectId = req.params.id;
    const dir = path.join(path.resolve(sharedDir), projectId, 'thumbnail');
    const candidates = THUMBNAIL_FILENAMES.map((name) => path.join(dir, name));

    const tryNext = (index) => {
      if (index >= candidates.length) {
        res.status(200).json({ exists: false, thumbnail_path: null, auto_generated: false });
        return;
      }
      fs.stat(candidates[index], (err, stat) => {
        if (err || !stat.isFile()) {
          tryNext(index + 1);
          return;
        }
        res.status(200).json({
          exists: true,
          thumbnail_path: candidates[index],
          // Lets the GUI say the image is a suggestion rather than the
          // Creator's own choice, so they know they can replace it.
          auto_generated: path.basename(candidates[index]) === 'auto.jpg',
        });
      });
    };
    tryNext(0);
  };
}

module.exports = { thumbnailUploadHandler, thumbnailServeHandler, thumbnailInfoHandler };
