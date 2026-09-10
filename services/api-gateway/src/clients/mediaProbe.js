'use strict';

const { execFile } = require('child_process');

/**
 * Thin wrapper over `ffprobe`, kept in its own module (rather than inline in
 * the handler) for the same reason `httpClient` is separate: the handler's
 * upload-time validation rules (CR-023 FR66.7 — 16:9, ≤5s) are business
 * logic worth unit-testing, while spawning a binary that may not exist in
 * the test image is infrastructure worth stubbing.
 *
 * @param {string} filePath - absolute path of a file already on disk.
 * @returns {Promise<{ width: number, height: number, durationSeconds: number }>}
 *   Rejects if ffprobe is missing, fails, or the file carries no video stream.
 */
function probeVideo(filePath) {
  return new Promise((resolve, reject) => {
    execFile(
      'ffprobe',
      [
        '-v',
        'error',
        '-select_streams',
        'v:0',
        '-show_entries',
        'stream=width,height:format=duration',
        '-of',
        'json',
        filePath,
      ],
      (err, stdout) => {
        if (err) {
          reject(new Error(`ffprobe failed: ${err.message}`));
          return;
        }
        let parsed;
        try {
          parsed = JSON.parse(stdout);
        } catch (parseErr) {
          reject(new Error(`ffprobe returned unreadable output: ${parseErr.message}`));
          return;
        }
        const stream = (parsed.streams || [])[0];
        if (!stream || !stream.width || !stream.height) {
          reject(new Error('file has no video stream'));
          return;
        }
        const durationSeconds = Number((parsed.format || {}).duration);
        if (!Number.isFinite(durationSeconds)) {
          reject(new Error('file has no readable duration'));
          return;
        }
        resolve({ width: Number(stream.width), height: Number(stream.height), durationSeconds });
      },
    );
  });
}

module.exports = { probeVideo };
