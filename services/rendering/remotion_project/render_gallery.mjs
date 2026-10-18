#!/usr/bin/env node
/**
 * Preview the illustration kit without running the pipeline.
 *
 *   node render_gallery.mjs sheets <outDir> [--kit key]   two PNG sheets: every pose/mood of
 *                                                         <Person>, and every other figure
 *   node render_gallery.mjs sample <outDir> [--kit key]   the sample scenes of the visual
 *                                                         language (fig interior, outer space),
 *                                                         landscape and portrait, as PNG stills
 *
 * Set REMOTION_BROWSER to a Chrome/Chromium headless shell to skip Remotion's
 * own browser download (e.g. on a machine that already has Playwright's).
 */
import {bundle} from '@remotion/bundler';
import {renderStill, selectComposition} from '@remotion/renderer';
import {mkdirSync} from 'node:fs';
import {fileURLToPath} from 'node:url';

const args = process.argv.slice(2);
const kitAt = args.indexOf('--kit');
const kitKey = kitAt >= 0 ? args.splice(kitAt, 2)[1] : 'phatsang';
const [mode, out] = args;
if (!['sheets', 'sample'].includes(mode) || !out) {
  console.error('usage: node render_gallery.mjs sheets|sample <outDir> [--kit key]');
  process.exit(2);
}
const browserExecutable = process.env.REMOTION_BROWSER || null;
const ENTRIES = {sheets: './src/gallery/index.tsx', sample: './src/gallery/language-sample-entry.tsx'};
const IDS = {sheets: ['people', 'things'], sample: ['sample-fig-wide', 'sample-fig-tall', 'sample-space-wide', 'sample-space-tall']};
const entry = fileURLToPath(new URL(ENTRIES[mode], import.meta.url));
const serveUrl = await bundle({entryPoint: entry, publicDir: fileURLToPath(new URL('./public', import.meta.url))});
const inputProps = {kitKey};

mkdirSync(out, {recursive: true});
for (const id of IDS[mode]) {
  const composition = await selectComposition({serveUrl, id, inputProps, browserExecutable});
  await renderStill({composition, serveUrl, frame: 10, inputProps, output: `${out}/${id}.png`, browserExecutable,
    chromiumOptions: {enableMultiProcessOnLinux: true, gl: 'swangle'}});
  console.log(`${out}/${id}.png`);
}
