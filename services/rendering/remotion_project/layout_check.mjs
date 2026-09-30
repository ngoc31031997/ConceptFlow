#!/usr/bin/env node
/**
 * Long-lived layout checker. The compile check
 * (services/rendering/adapters/rendering/layout_checker.py) starts it once and
 * keeps it: the harness bundle and the browser are paid for at start-up, and
 * each check then costs a fresh page, the script's compile and the measuring.
 *
 * It only measures (layout_probe_lib.mjs); the rules are applied in Python.
 *
 * Protocol: one JSON object per line on stdin, one per line on stdout.
 *   at start:  {"ready": true, "browser": {"version"}}
 *          or: {"fatal": "..."}                         (then it exits 1)
 *   in:        {"id": "...", "code": "<tsx>", "font": "Be Vietnam Pro"}
 *   out:       {"id": "...", "result": {...probe JSON, version 1...}}
 *          or: {"id": "...", "error": "...", "stage": "compile|load|measure|browser"}
 * A request whose browser has died is answered with stage "browser" and the
 * process exits, so the caller starts a new one.
 */
import readline from 'node:readline';
import {DEFAULT_DURATION, DEFAULT_PCTS, ProbeError, bundleHarness, launchBrowser, openPage, probe} from './layout_probe_lib.mjs';

const write = (obj) => process.stdout.write(JSON.stringify(obj) + '\n');

let browser;
let harnessJs;
try {
  harnessJs = await bundleHarness();
  browser = await launchBrowser();
} catch (err) {
  write({fatal: String(err?.message ?? err).slice(0, 1500)});
  process.exit(1);
}
let closing = false;
browser.on('disconnected', () => {
  if (!closing) process.stderr.write('layout_check: the browser disconnected\n');
});
write({ready: true, browser: {version: browser.version()}});

async function handle(req) {
  const t0 = performance.now();
  if (!browser.isConnected()) throw new ProbeError('browser', 'the browser is no longer running');
  let opened;
  try {
    opened = await openPage(browser, harnessJs);
  } catch (err) {
    throw new ProbeError('browser', `could not open a page: ${String(err?.message ?? err).split('\n')[0]}`);
  }
  try {
    const result = await probe(opened.page, opened.logs, String(req.code ?? ''), {
      duration: DEFAULT_DURATION,
      pcts: DEFAULT_PCTS,
      font: req.font || null,
    });
    result.browser = {version: browser.version()};
    result.timing_ms.total = Math.round(performance.now() - t0);
    return result;
  } finally {
    await opened.page.close().catch(() => {});
  }
}

// One request at a time: the caller serialises, and one page keeps memory flat.
let chain = Promise.resolve();
const rl = readline.createInterface({input: process.stdin, crlfDelay: Infinity});
rl.on('line', (line) => {
  if (!line.trim()) return;
  chain = chain.then(async () => {
    let id = null;
    try {
      const req = JSON.parse(line);
      id = req.id ?? null;
      write({id, result: await handle(req)});
    } catch (err) {
      const stage = err instanceof ProbeError ? err.stage : browser.isConnected() ? 'measure' : 'browser';
      write({id, error: String(err?.message ?? err).slice(0, 2000), stage});
      if (stage === 'browser' || !browser.isConnected()) process.exit(1);
    }
  });
});
rl.on('close', () => {
  closing = true;
  chain.then(() => browser.close().catch(() => {})).then(() => process.exit(0));
});
