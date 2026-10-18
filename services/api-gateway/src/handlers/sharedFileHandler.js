'use strict';

const fs = require('fs');
const path = require('path');

/**
 * Serves one file of a directory on the shared volume, by name, with HTTP
 * Range support. Only names that match `namePattern` are served, and the
 * resolved path must stay inside `dir`, so a request can never read anything
 * but what the owning service wrote there.
 *
 * @param {string} dir - absolute directory the files live in, e.g. "/shared/stills"
 * @param {{param: string, namePattern: RegExp, contentType: string}} options
 *   `param` is the route parameter holding the file name.
 */
function sharedFileHandler(dir, { param, namePattern, contentType }) {
  const root = path.resolve(dir);
  return (req, res) => {
    const name = req.params[param];
    if (typeof name !== 'string' || !namePattern.test(name)) {
      res.status(404).json({ error: 'file_not_found' });
      return;
    }
    const file = path.resolve(root, name);
    if (!file.startsWith(root + path.sep)) {
      res.status(404).json({ error: 'file_not_found' });
      return;
    }

    fs.stat(file, (statErr, stat) => {
      if (statErr || !stat.isFile()) {
        res.status(404).json({ error: 'file_not_found' });
        return;
      }
      const range = req.headers.range;
      if (!range) {
        res.writeHead(200, {
          'Content-Type': contentType,
          'Content-Length': stat.size,
          'Accept-Ranges': 'bytes',
          // A sample is named by the hash of what it shows, so it never changes.
          'Cache-Control': 'public, max-age=31536000, immutable',
        });
        fs.createReadStream(file).pipe(res);
        return;
      }

      const match = /^bytes=(\d*)-(\d*)$/.exec(range);
      const start = match && match[1] !== '' ? parseInt(match[1], 10) : NaN;
      const end = match && match[2] !== '' ? parseInt(match[2], 10) : stat.size - 1;
      if (!match || Number.isNaN(start) || start > end || start >= stat.size) {
        res.writeHead(416, { 'Content-Range': `bytes */${stat.size}` });
        res.end();
        return;
      }
      const last = Math.min(end, stat.size - 1);
      res.writeHead(206, {
        'Content-Range': `bytes ${start}-${last}/${stat.size}`,
        'Accept-Ranges': 'bytes',
        'Content-Length': last - start + 1,
        'Content-Type': contentType,
      });
      fs.createReadStream(file, { start, end: last }).pipe(res);
    });
  };
}

module.exports = { sharedFileHandler };
