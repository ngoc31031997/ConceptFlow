// Bundles a kit source file with esbuild and imports it, so tests run the same
// TypeScript the Remotion bundle compiles.
import assert from 'node:assert/strict';
import {mkdirSync} from 'node:fs';
import path from 'node:path';
import {pathToFileURL, fileURLToPath} from 'node:url';
import {build} from 'esbuild';

const here = path.dirname(fileURLToPath(import.meta.url));
const kit = path.join(here, '..', 'src', 'conceptflow-mini');

/** Imports `src/conceptflow-mini/<entry>` as an ES module. */
export async function load(entry) {
  // Inside node_modules so the externals ('react', 'remotion') resolve.
  const dir = path.join(here, '..', 'node_modules', '.kit-test');
  mkdirSync(dir, {recursive: true});
  const out = path.join(dir, entry.replace(/\W/g, '_') + '.mjs');
  await build({
    entryPoints: [path.join(kit, entry)],
    outfile: out,
    bundle: true,
    format: 'esm',
    platform: 'node',
    jsx: 'automatic',
    external: ['react', 'react/jsx-runtime', 'remotion', '@remotion/*'],
    logLevel: 'silent',
  });
  return import(pathToFileURL(out).href);
}

/** Asserts two numbers differ by less than `eps`. */
export function near(a, b, eps = 1e-9) {
  assert.ok(Math.abs(a - b) < eps, `${a} != ${b}`);
}
