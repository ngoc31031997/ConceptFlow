#!/usr/bin/env node
/**
 * Preview the flat illustration kit (CR-043) without running the pipeline.
 *
 *   node render_gallery.mjs sheets <outDir>   two PNG sheets: every pose/mood of
 *                                             <Person>, and every other figure
 *   node render_gallery.mjs demo <out.mp4>    the 16-second "tooth decay" sample
 *                                             (src/gallery/tooth-decay-demo.tsx)
 *
 * Set REMOTION_BROWSER to a Chrome/Chromium headless shell to skip Remotion's
 * own browser download (e.g. on a machine that already has Playwright's).
 */
import {bundle} from '@remotion/bundler';
import {renderMedia, renderStill, selectComposition} from '@remotion/renderer';
import {mkdirSync} from 'node:fs';
import {fileURLToPath} from 'node:url';

const [mode, out] = process.argv.slice(2);
if (!['sheets', 'demo'].includes(mode) || !out) {
  console.error('usage: node render_gallery.mjs sheets <outDir> | demo <out.mp4>');
  process.exit(2);
}
const browserExecutable = process.env.REMOTION_BROWSER || null;
const entry = fileURLToPath(new URL(mode === 'demo' ? './src/gallery/tooth-decay-demo.tsx' : './src/gallery/index.tsx', import.meta.url));
const serveUrl = await bundle({entryPoint: entry, publicDir: fileURLToPath(new URL('./public', import.meta.url))});

if (mode === 'sheets') {
  mkdirSync(out, {recursive: true});
  for (const id of ['people', 'things']) {
    const composition = await selectComposition({serveUrl, id, inputProps: {}, browserExecutable});
    await renderStill({composition, serveUrl, frame: 10, output: `${out}/${id}.png`, browserExecutable});
    console.log(`${out}/${id}.png`);
  }
} else {
  const composition = await selectComposition({serveUrl, id: 'creator', inputProps: {}, browserExecutable});
  await renderMedia({composition, serveUrl, codec: 'h264', outputLocation: out, browserExecutable,
    chromiumOptions: {enableMultiProcessOnLinux: true}});
  console.log(out);
}
