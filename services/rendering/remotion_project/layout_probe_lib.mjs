/**
 * Layout probe — the measuring half (CR-048 T6a spike, wired in by T6b).
 *
 * Measures, in headless Chromium, the DOM boxes of what each shot of a merged
 * Remotion script draws at chosen moments of the shot. It only MEASURES: the
 * layout rules (safe area, subtitle zone, text overflow, text over text, tiny
 * font, hero size) and their Vietnamese messages live in Python
 * (services/rendering/domain/layout_rules.py), where they are tested without a
 * browser.
 *
 * Used by:
 *   - layout_check.mjs — the long-lived checker the compile check talks to;
 *   - layout_probe.mjs — the command line tool (writes the JSON to look at).
 *
 * How a script is turned into something measurable:
 *   - TypeScript transpiles it to CommonJS with one extra transform: every JSX
 *     element gets `data-cf-line={its line}`, so every DOM box can be traced
 *     to the JSX line (and so to the shot section) that drew it;
 *   - src/layout-probe/harness.tsx (bundled by esbuild, once per process)
 *     evaluates it and draws the registered composition with
 *     @remotion/player's <Thumbnail>.
 */
import {build} from 'esbuild';
import {chromium} from 'playwright-core';
import ts from 'typescript';
import {readFileSync, existsSync} from 'node:fs';
import {join, resolve, extname} from 'node:path';
import {fileURLToPath} from 'node:url';

const HERE = fileURLToPath(new URL('.', import.meta.url));
const ORIGIN = 'http://cf-layout-probe.local';

/** How long one sample waits for delayRender() handles (font, Lottie JSON). */
export const SETTLE_TIMEOUT_MS = 5000;

/** Nominal shot length: at the code step there is no TTS yet (5 s at 30 fps). */
export const DEFAULT_DURATION = 150;

/**
 * The moments of each shot that are measured, as fractions of the shot
 * (frame = round(pct × (duration − 1))).
 *
 * Every 5 frames (1/6 s) through the first second, where entry animations
 * run: a spring's first overshoot peaks about 10 frames after it starts
 * (stiffness 100, the default), and the T6a spike showed 4 samples missing it
 * while denser ones catch it. Then 1.5 s, the middle, 70 %, and the settled
 * part (85 %, the last frame) where the layout rules for text size apply.
 * 12 samples ≈ 11 ms each: about 4 s for a 30-shot video.
 */
export const DEFAULT_PCTS = [0, 5, 10, 15, 20, 25, 30, 45, 75, 104, 127, 149].map((f) => f / 149);

// --- compile -------------------------------------------------------------------------

/** TS transformer: prepend data-cf-line={line} to every JSX element. Prepended,
 * so a `{...props}` spread that carries the caller's line (a library drawing
 * passing its props to <Figure>) wins over the line inside the drawing. */
function lineTagger(context) {
  const f = context.factory;
  return (sf) => {
    const visit = (node) => {
      if (ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) {
        const line = sf.getLineAndCharacterOfPosition(node.getStart(sf)).line + 1;
        const attr = f.createJsxAttribute(f.createIdentifier('data-cf-line'), f.createJsxExpression(undefined, f.createNumericLiteral(line)));
        const attrs = f.updateJsxAttributes(node.attributes, [attr, ...node.attributes.properties]);
        node = ts.isJsxOpeningElement(node)
          ? f.updateJsxOpeningElement(node, node.tagName, node.typeArguments, attrs)
          : f.updateJsxSelfClosingElement(node, node.tagName, node.typeArguments, attrs);
      }
      return ts.visitEachChild(node, visit, context);
    };
    return ts.visitNode(sf, visit);
  };
}

export function compile(source, fileName = 'script.tsx') {
  const out = ts.transpileModule(source, {
    fileName,
    reportDiagnostics: true,
    compilerOptions: {
      jsx: ts.JsxEmit.ReactJSX,
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2020,
      esModuleInterop: true,
    },
    transformers: {before: [lineTagger]},
  });
  const errors = (out.diagnostics ?? []).filter((d) => d.category === ts.DiagnosticCategory.Error);
  if (errors.length) {
    const msg = errors.map((d) => {
      const pos = d.file && d.start !== undefined ? d.file.getLineAndCharacterOfPosition(d.start) : null;
      return `${pos ? `${pos.line + 1}:${pos.character + 1}: ` : ''}${ts.flattenDiagnosticMessageText(d.messageText, '\n')}`;
    });
    throw new ProbeError('compile', `syntax error: ${msg.join('; ')}`);
  }
  return out.outputText;
}

/** The placeholder llm-service's merger writes for a shot another chunk owns
 * (merger.remotion_stub): it draws nothing, so a one-chunk check skips it. */
const STUB = /^function\s+\w+\s*\(\s*\{\s*duration\s*\}\s*:\s*ShotProps\s*\)\s*\{\s*return\s+null;\s*\}\s*$/;

/** Shot order and declaration lines, read from the merged file the same way the
 * merger writes it: `const SHOTS ... = [Shot1_1, ...]` and `function ShotN_M(`. */
export function shotTable(source) {
  const lines = source.split('\n');
  const m = /const\s+SHOTS\b[^=]*=\s*\[([^\]]*)\]/.exec(source);
  if (!m) return null;
  const shotsLine = source.slice(0, m.index).split('\n').length;
  const names = m[1].split(',').map((s) => s.trim()).filter(Boolean);
  const shots = names.map((name, index) => {
    const decl = new RegExp(`^\\s*(?:export\\s+)?(?:function\\s+${name}\\s*\\(|(?:const|let|var)\\s+${name}\\b)`);
    const at = lines.findIndex((l) => decl.test(l));
    const id = /^Shot(\d+)_(\d+)$/.exec(name);
    return {index, component: name, id: id ? `${id[1]}.${id[2]}` : null, line: at >= 0 ? at + 1 : null};
  });
  // A shot's text runs from its declaration to the next one (or `const SHOTS`);
  // the next shot's own `// Shot N.M — ...` comment lines sit at its end.
  const starts = shots.map((s) => s.line).filter(Boolean).sort((a, b) => a - b);
  for (const s of shots) {
    if (!s.line) continue;
    const end = starts.find((l) => l > s.line) ?? shotsLine;
    const body = lines.slice(s.line - 1, end - 1).filter((l) => !/^\s*\/\//.test(l)).join('\n').trim();
    s.stub = STUB.test(body);
  }
  return {shots_line: shotsLine, shots};
}

/** A failure with the stage it happened in: `compile` and `load` are the
 * script's own (it does not evaluate), `browser` and `measure` the probe's. */
export class ProbeError extends Error {
  constructor(stage, message) {
    super(message);
    this.stage = stage;
  }
}

// --- page ---------------------------------------------------------------------------

export async function bundleHarness() {
  const shim = join(HERE, 'src/layout-probe/remotion-shim.ts');
  const result = await build({
    entryPoints: [join(HERE, 'src/layout-probe/harness.tsx')],
    bundle: true,
    write: false,
    format: 'iife',
    platform: 'browser',
    target: 'chrome120',
    jsx: 'automatic',
    define: {'process.env.NODE_ENV': '"production"'},
    logLevel: 'silent',
    plugins: [
      {
        name: 'remotion-shim',
        setup(b) {
          b.onResolve({filter: /^remotion$/}, (args) => (args.importer === shim ? undefined : {path: shim}));
        },
      },
    ],
  });
  return result.outputFiles[0].text;
}

/**
 * The browser: `explicit` (--browser), else $LAYOUT_PROBE_BROWSER, else the
 * Chromium headless shell Playwright manages for this playwright-core version
 * (installed into $PLAYWRIGHT_BROWSERS_PATH by the rendering Dockerfile). Never
 * downloads anything: a missing browser is an error the caller reports.
 */
export async function launchBrowser(explicit) {
  const executablePath = explicit || process.env.LAYOUT_PROBE_BROWSER || undefined;
  if (executablePath && !existsSync(executablePath)) {
    throw new ProbeError('browser', `browser not found at ${executablePath}`);
  }
  try {
    return await chromium.launch({executablePath, headless: true, args: ['--disable-dev-shm-usage']});
  } catch (err) {
    throw new ProbeError('browser', `could not start Chromium: ${String(err?.message ?? err).split('\n')[0]}`);
  }
}

/** A fresh page serving the harness, with every request outside the harness
 * origin aborted. `logs` collects page errors and blocked requests. */
export async function openPage(browser, harnessJs) {
  const page = await browser.newPage({viewport: {width: 1920, height: 1080}, deviceScaleFactor: 1});
  const logs = [];
  page.on('pageerror', (e) => logs.push({type: 'pageerror', text: String(e?.message ?? e).slice(0, 1000)}));
  page.on('console', (m) => {
    if (m.type() === 'error' || m.type() === 'warning') logs.push({type: m.type(), text: m.text().slice(0, 1000)});
  });
  const html = `<!doctype html><html><head><meta charset="utf-8"><style>html,body{margin:0;background:#000}</style></head>` +
    `<body><div id="cf-stage" style="position:relative;width:1920px;height:1080px"></div><script src="/harness.js"></script></body></html>`;
  const publicDir = join(HERE, 'public');
  // The script is untrusted model output: nothing leaves the page. (Playwright
  // tries the most recently added route first, so this one is the fallback.)
  await page.route('**/*', (route) => {
    logs.push({type: 'blocked', text: route.request().url().slice(0, 200)});
    return route.abort();
  });
  await page.route(`${ORIGIN}/**`, async (route) => {
    const url = new URL(route.request().url());
    if (url.pathname === '/' || url.pathname === '/index.html') return route.fulfill({contentType: 'text/html', body: html});
    if (url.pathname === '/harness.js') return route.fulfill({contentType: 'application/javascript', body: harnessJs});
    // staticFile() — the Lottie catalog lives in public/, as at render time.
    const file = resolve(publicDir, '.' + decodeURIComponent(url.pathname));
    if (file.startsWith(publicDir) && existsSync(file)) {
      return route.fulfill({contentType: extname(file) === '.json' ? 'application/json' : 'application/octet-stream', body: readFileSync(file)});
    }
    logs.push({type: 'http404', text: url.pathname});
    return route.fulfill({status: 404, body: 'not found'});
  });
  await page.goto(`${ORIGIN}/index.html`);
  await page.waitForFunction(() => Boolean(window.cfProbe));
  return {page, logs};
}

// --- measuring --------------------------------------------------------------------------

const firstLine = (err) => String(err?.message ?? err).split('\n')[0].slice(0, 1000);

/**
 * Measures one merged script on `page` (a page from openPage; the caller gives
 * each script a fresh page so nothing a script leaves on `window` reaches the
 * next one). Returns the probe JSON (version 1, see the T6a findings, §7).
 *
 * opts: {duration, pcts, font}
 */
export async function probe(page, logs, source, opts) {
  const t = {};
  let t0 = performance.now();
  const js = compile(source);
  t.compile = Math.round(performance.now() - t0);
  const table = shotTable(source);

  t0 = performance.now();
  const inputProps = opts.font ? {videoFont: opts.font} : {};
  let info;
  try {
    info = await page.evaluate(([code, props]) => window.cfProbe.load(code, props), [js, inputProps]);
  } catch (err) {
    throw new ProbeError('load', `the script did not load: ${firstLine(err)}`);
  }
  t.load = Math.round(performance.now() - t0);

  const count = table ? table.shots.length : info.narrations;
  if (!count) throw new ProbeError('load', 'cannot tell how many shots the script has (no SHOTS array, no narrations)');
  const font = await page.evaluate((f) => window.cfProbe.fontStatus(f), opts.font || 'Be Vietnam Pro');
  t0 = performance.now();
  let measured;
  try {
    measured = await page.evaluate(
      (o) => window.cfProbe.measure(o),
      {
        shots: count, duration: opts.duration, pcts: opts.pcts, settleTimeoutMs: SETTLE_TIMEOUT_MS,
        skip: (table?.shots ?? []).filter((s) => s.stub).map((s) => s.index),
      },
    );
  } catch (err) {
    throw new ProbeError('measure', `measuring failed: ${firstLine(err)}`);
  }
  t.measure = Math.round(performance.now() - t0);

  const shots = measured.map((m) => ({
    ...(table?.shots[m.index] ?? {index: m.index, component: null, id: null, line: null}),
    samples: m.samples,
    ...(m.skipped ? {skipped: m.skipped} : {}),
  }));
  const sampleMs = shots.flatMap((s) => s.samples.map((x) => x.ms));
  return {
    version: 1,
    composition: {id: info.id, width: info.width, height: info.height, fps: info.fps},
    narrations: info.narrations,
    nominal_duration: opts.duration,
    frames: opts.pcts.map((p) => ({pct: p, frame: Math.round(p * (opts.duration - 1))})),
    font,
    shots_line: table?.shots_line ?? null,
    shots,
    page_log: [...logs],
    timing_ms: {
      ...t,
      samples: sampleMs.length,
      sample_avg: Math.round((sampleMs.reduce((a, b) => a + b, 0) / (sampleMs.length || 1)) * 10) / 10,
      sample_max: Math.max(0, ...sampleMs),
    },
  };
}
