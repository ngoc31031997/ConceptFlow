// Pure-logic tests of the kit (light direction, colour maths, rig maths...).
// Run: node --test kit_tests/   (from remotion_project, after npm install).
// The TypeScript sources are bundled with esbuild into a temp file first, so the
// tests exercise the same code the Remotion bundle compiles.
import assert from 'node:assert/strict';
import {mkdirSync} from 'node:fs';
import path from 'node:path';
import {pathToFileURL, fileURLToPath} from 'node:url';
import {test} from 'node:test';
import {build} from 'esbuild';

const here = path.dirname(fileURLToPath(import.meta.url));
const kit = path.join(here, '..', 'src', 'conceptflow-mini');

async function load(entry) {
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

const near = (a, b, eps = 1e-9) => assert.ok(Math.abs(a - b) < eps, `${a} != ${b}`);

test('light direction points from the figure to the light source', async () => {
  const {lightDirection} = await load('material.tsx');
  const frame = {x: 500, y: 500, rotate: 0, flip: false};
  const [dx, dy] = lightDirection(frame, [900, 500]);
  near(dx, 1);
  near(dy, 0);
  const [ux, uy] = lightDirection(frame, [500, 100]);
  near(ux, 0);
  near(uy, -1);
});

test('a mirrored figure sees the light on the other side of its drawing', async () => {
  const {lightDirection} = await load('material.tsx');
  const [dx] = lightDirection({x: 500, y: 500, rotate: 0, flip: true}, [900, 500]);
  near(dx, -1);
});

test('a rotated figure turns the light direction the opposite way', async () => {
  const {lightDirection} = await load('material.tsx');
  const [dx, dy] = lightDirection({x: 0, y: 0, rotate: 90, flip: false}, [0, -100]);
  near(dx, -1);
  near(dy, 0);
});

test('a light on the figure itself shines from above', async () => {
  const {lightDirection} = await load('material.tsx');
  assert.deepEqual(lightDirection({x: 10, y: 10, rotate: 0, flip: false}, [10, 10]), [0, -1]);
});

test('a capsule path starts and ends on its own outline', async () => {
  const {capsulePath} = await load('material.tsx');
  const d = capsulePath({x1: 0, y1: 0, x2: 100, y2: 0, r: 10});
  assert.match(d, /^M 0 10 L 100 10 A 10 10 0 0 0 100 -10 L 0 -10 A 10 10 0 0 0 0 10 Z$/);
});

test('colour helpers blend, darken and reject non-hex notation', async () => {
  const {mix, darken, withAlpha, parseHex} = await load('film.tsx');
  assert.equal(mix('#000000', '#ffffff', 0.5), '#808080');
  assert.equal(darken('#ff8040', 0.5), '#804020');
  assert.equal(withAlpha('#ff0000', 0.5), '#ff000080');
  assert.deepEqual(parseHex('#abc'), [170, 187, 204]);
  assert.throws(() => parseHex('red'), /không phải dạng/);
});
