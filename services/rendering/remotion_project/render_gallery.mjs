#!/usr/bin/env node
/**
 * Preview the flat illustration kit without running the pipeline.
 *
 *   node render_gallery.mjs sheets <outDir>   two PNG sheets: every pose/mood of
 *                                             <Person>, and every other figure
 *   node render_gallery.mjs demo <out.mp4>    the 16-second "tooth decay" sample
 *                                             (src/gallery/tooth-decay-demo.tsx)
 *   node render_gallery.mjs scene <outDir>    the scene-kit sample, landscape and
 *                                             portrait (src/gallery/scene-demo.tsx)
 *
 * Set REMOTION_BROWSER to a Chrome/Chromium headless shell to skip Remotion's
 * own browser download (e.g. on a machine that already has Playwright's).
 */
import {bundle} from '@remotion/bundler';
import {renderMedia, renderStill, selectComposition} from '@remotion/renderer';
import {mkdirSync} from 'node:fs';
import {fileURLToPath} from 'node:url';

const [mode, out] = process.argv.slice(2);
if (!['sheets', 'demo', 'scene'].includes(mode) || !out) {
  console.error('usage: node render_gallery.mjs sheets <outDir> | demo <out.mp4> | scene <outDir>');
  process.exit(2);
}
const browserExecutable = process.env.REMOTION_BROWSER || null;
const ENTRIES = {sheets: './src/gallery/index.tsx', demo: './src/gallery/tooth-decay-demo.tsx', scene: './src/gallery/scene-demo.tsx'};
const entry = fileURLToPath(new URL(ENTRIES[mode], import.meta.url));
const serveUrl = await bundle({entryPoint: entry, publicDir: fileURLToPath(new URL('./public', import.meta.url))});

if (mode === 'sheets') {
  mkdirSync(out, {recursive: true});
  for (const id of ['people', 'things']) {
    const composition = await selectComposition({serveUrl, id, inputProps: {}, browserExecutable});
    await renderStill({composition, serveUrl, frame: 10, output: `${out}/${id}.png`, browserExecutable});
    console.log(`${out}/${id}.png`);
  }
} else if (mode === 'scene') {
  mkdirSync(out, {recursive: true});
  for (const id of ['scene-wide', 'scene-tall']) {
    const composition = await selectComposition({serveUrl, id, inputProps: {}, browserExecutable});
    await renderMedia({composition, serveUrl, codec: 'h264', outputLocation: `${out}/${id}.mp4`, browserExecutable,
      chromiumOptions: {enableMultiProcessOnLinux: true}});
    console.log(`${out}/${id}.mp4`);
  }
} else {
  const composition = await selectComposition({serveUrl, id: 'creator', inputProps: {}, browserExecutable});
  await renderMedia({composition, serveUrl, codec: 'h264', outputLocation: out, browserExecutable,
    chromiumOptions: {enableMultiProcessOnLinux: true}});
  console.log(out);
}
