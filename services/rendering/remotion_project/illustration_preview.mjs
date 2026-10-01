#!/usr/bin/env node
/**
 * Long-lived preview renderer for the illustration library.
 *
 * Bundles src/illustration-preview/host.tsx and opens the browser once, then
 * answers one JSON request per stdin line with one JSON reply per stdout line:
 *
 *   -> {"id", "code"?: TSX source, "name": component, "props"?: {...}, "gif"?: bool, "kind"?: "figure" or "backdrop"}
 *   <- {"id", "ok": true, "png": base64, "gif"?: base64}
 *   <- {"id", "ok": false, "error": "..."}
 *
 * `code` absent means a built-in kit component. The TSX is compiled here with
 * esbuild (CommonJS, automatic JSX) and evaluated by the host inside the page.
 * Rendering the same frame twice gives the same pixels: every motion in a
 * figure is driven by the frame number.
 */
import {bundle} from '@remotion/bundler';
import {openBrowser, renderMedia, renderStill, selectComposition} from '@remotion/renderer';
import {transform} from 'esbuild';
import {readFileSync, rmSync, mkdtempSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {createInterface} from 'node:readline';
import {fileURLToPath} from 'node:url';

const browserExecutable = process.env.REMOTION_BROWSER || null;
const entry = fileURLToPath(new URL('./src/illustration-preview/host.tsx', import.meta.url));
const serveUrl = await bundle({entryPoint: entry});
const browser = await openBrowser('chrome', {browserExecutable, chromiumOptions: {enableMultiProcessOnLinux: true}});
const out = (msg) => process.stdout.write(JSON.stringify(msg) + '\n');
out({ready: true});

async function handle(req) {
  let js = '';
  if (req.code) {
    try {
      js = (await transform(req.code, {loader: 'tsx', format: 'cjs', jsx: 'automatic', sourcefile: `${req.name}.tsx`})).code;
    } catch (err) {
      const msgs = (err.errors ?? []).map((e) => `${e.location?.line}:${e.location?.column}: ${e.text}`);
      throw new Error(`syntax error: ${msgs.join('; ') || err.message}`);
    }
  }
  const inputProps = {js, name: req.name, props: req.props ?? {}, background: req.background, kind: req.kind ?? 'figure'};
  const composition = await selectComposition({serveUrl, id: 'asset', inputProps, puppeteerInstance: browser});
  const dir = mkdtempSync(join(tmpdir(), 'asset-'));
  try {
    const png = join(dir, 'still.png');
    await renderStill({composition, serveUrl, inputProps, frame: 20, output: png, puppeteerInstance: browser});
    const reply = {id: req.id, ok: true, png: readFileSync(png).toString('base64')};
    if (req.gif) {
      const gif = join(dir, 'clip.gif');
      await renderMedia({composition, serveUrl, inputProps, codec: 'gif', outputLocation: gif, scale: 0.4,
        everyNthFrame: 3, puppeteerInstance: browser});
      reply.gif = readFileSync(gif).toString('base64');
    }
    return reply;
  } finally {
    rmSync(dir, {recursive: true, force: true});
  }
}

// One request at a time: the caller serialises, and one browser tab keeps memory flat.
let chain = Promise.resolve();
createInterface({input: process.stdin}).on('line', (line) => {
  chain = chain.then(async () => {
    let req;
    try {
      req = JSON.parse(line);
      out(await handle(req));
    } catch (err) {
      out({id: req?.id ?? null, ok: false, error: String(err?.message ?? err).slice(0, 4000)});
    }
  });
}).on('close', () => chain.then(() => browser.close({silent: true})).then(() => process.exit(0)));
