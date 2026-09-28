#!/usr/bin/env node
/**
 * Layout probe, command line (CR-048 T6a; the compile check uses
 * layout_check.mjs, the same measuring code kept warm).
 *
 * Writes the measured DOM boxes of every shot of one or more merged Remotion
 * scripts as JSON — to look at what the checker sees, or to record a test
 * fixture. It applies no rules: those live in services/rendering/domain/layout_rules.py.
 *
 *   node layout_probe.mjs [options] <merged.tsx> [<merged.tsx> ...]
 *
 * Options:
 *   --duration N      nominal frames per shot (default 150 = 5 s at 30 fps)
 *   --frames a,b,..   moments as fractions of the shot (default: the compile
 *                     check's own list, DEFAULT_PCTS in layout_probe_lib.mjs;
 *                     1 = the shot's last frame, duration - 1)
 *   --font NAME       videoFont input prop (default: the Stage's own default)
 *   --out-dir DIR     write <name>.layout.json per script there (else stdout)
 *   --browser PATH    Chromium executable; else $LAYOUT_PROBE_BROWSER, else
 *                     Playwright's own headless shell
 */
import {mkdirSync, readFileSync, writeFileSync} from 'node:fs';
import {basename, join} from 'node:path';
import {DEFAULT_DURATION, DEFAULT_PCTS, bundleHarness, launchBrowser, openPage, probe} from './layout_probe_lib.mjs';

function parseArgs(argv) {
  const opts = {duration: DEFAULT_DURATION, pcts: DEFAULT_PCTS, font: null, outDir: null, browser: null, scripts: []};
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    const next = () => argv[++i];
    if (a === '--duration') opts.duration = Number(next());
    else if (a === '--frames') opts.pcts = next().split(',').map(Number);
    else if (a === '--font') opts.font = next();
    else if (a === '--out-dir') opts.outDir = next();
    else if (a === '--browser') opts.browser = next();
    else if (a.startsWith('--')) throw new Error(`unknown option ${a}`);
    else opts.scripts.push(a);
  }
  if (!opts.scripts.length) throw new Error('usage: node layout_probe.mjs [options] <merged.tsx> ...');
  if (!(opts.duration >= 2)) throw new Error('--duration must be >= 2');
  if (opts.pcts.some((p) => !(p >= 0 && p <= 1))) throw new Error('--frames must be fractions in [0, 1]');
  return opts;
}

async function main() {
  const opts = parseArgs(process.argv.slice(2));
  const tStart = performance.now();
  let t0 = performance.now();
  const harnessJs = await bundleHarness();
  const tBundle = Math.round(performance.now() - t0);
  t0 = performance.now();
  const browser = await launchBrowser(opts.browser);
  const tBrowser = Math.round(performance.now() - t0);

  const results = [];
  try {
    for (const script of opts.scripts) {
      t0 = performance.now();
      const {page, logs} = await openPage(browser, harnessJs);
      const r = await probe(page, logs, readFileSync(script, 'utf-8'), opts);
      await page.close();
      r.script = script;
      r.timing_ms.script_total = Math.round(performance.now() - t0);
      r.browser = {version: browser.version()};
      r.timing_ms.startup = {bundle_harness: tBundle, launch_browser: tBrowser};
      results.push(r);
      process.stderr.write(`${script}: ${r.shots.length} shots × ${opts.pcts.length} frames in ${r.timing_ms.script_total} ms (measure ${r.timing_ms.measure} ms)\n`);
    }
  } finally {
    await browser.close();
  }
  process.stderr.write(`startup: bundle ${tBundle} ms, browser ${tBrowser} ms; total wall ${Math.round(performance.now() - tStart)} ms\n`);
  if (opts.outDir) {
    mkdirSync(opts.outDir, {recursive: true});
    for (const r of results) writeFileSync(join(opts.outDir, basename(r.script).replace(/\.tsx$/, '') + '.layout.json'), JSON.stringify(r, null, 2) + '\n');
  } else {
    process.stdout.write(JSON.stringify(results.length === 1 ? results[0] : results, null, 2) + '\n');
  }
}

main().catch((err) => {
  console.error(err?.stack ?? err);
  process.exit(1);
});
