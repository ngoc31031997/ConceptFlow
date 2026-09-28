#!/usr/bin/env node
/**
 * Layout probe — CR-048 T6a SPIKE. Not wired into any pipeline (that is T6b).
 *
 * Measures, in headless Chromium, the DOM boxes of what each shot of a merged
 * Remotion script draws at chosen moments of the shot, so layout rules (safe
 * area, text overflow, text over text, minimum font size) can be checked by
 * code instead of by the model's arithmetic.
 *
 *   node layout_probe.mjs [options] <merged.tsx> [<merged.tsx> ...]
 *
 * Options:
 *   --duration N      nominal frames per shot (default 150 = 5 s at 30 fps)
 *   --frames a,b,..   moments as fractions of the shot (default 0,0.5,0.85,1;
 *                     1 = the shot's last frame, duration - 1)
 *   --font NAME       videoFont input prop (default: the Stage's own default)
 *   --check           also apply the prototype rules and list violations
 *   --subtitle-band top:240|bottom:240   burned-in subtitle strip to keep clear
 *   --out-dir DIR     write <name>.layout.json per script there (else stdout)
 *   --browser PATH    Chromium/Chrome executable; else $LAYOUT_PROBE_BROWSER,
 *                     else Remotion's own Chrome Headless Shell (ensureBrowser,
 *                     the one the rendering image downloads at build time)
 *
 * One browser and one page serve every script given, the way a warm checker
 * process would; the JSON records how long each part took.
 *
 * How a script is turned into something measurable:
 *   - TypeScript transpiles it to CommonJS with one extra transform: every JSX
 *     element gets `data-cf-line={its line}`, so every DOM box can be traced
 *     to the JSX line (and so to the shot section) that drew it;
 *   - src/layout-probe/harness.tsx (bundled by esbuild, once) evaluates it and
 *     draws the registered composition with @remotion/player's <Thumbnail>.
 */
import {build} from 'esbuild';
import {chromium} from 'playwright-core';
import ts from 'typescript';
import {readFileSync, writeFileSync, mkdirSync, existsSync} from 'node:fs';
import {basename, join, resolve, extname} from 'node:path';
import {fileURLToPath} from 'node:url';

const HERE = fileURLToPath(new URL('.', import.meta.url));
const SAFE = {left: 96, top: 96, right: 1824, bottom: 984};
const MIN_FONT = 32;
const MIN_HERO_FRACTION = 0.3;
const ORIGIN = 'http://cf-layout-probe.local';

function parseArgs(argv) {
  const opts = {duration: 150, pcts: [0, 0.5, 0.85, 1], font: null, check: false, band: null, outDir: null, browser: null, scripts: []};
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    const next = () => argv[++i];
    if (a === '--duration') opts.duration = Number(next());
    else if (a === '--frames') opts.pcts = next().split(',').map(Number);
    else if (a === '--font') opts.font = next();
    else if (a === '--check') opts.check = true;
    else if (a === '--subtitle-band') {
      const [edge, px] = next().split(':');
      opts.band = {edge, px: Number(px)};
    } else if (a === '--out-dir') opts.outDir = next();
    else if (a === '--browser') opts.browser = next();
    else if (a.startsWith('--')) throw new Error(`unknown option ${a}`);
    else opts.scripts.push(a);
  }
  if (!opts.scripts.length) throw new Error('usage: node layout_probe.mjs [options] <merged.tsx> ...');
  if (!(opts.duration >= 2)) throw new Error('--duration must be >= 2');
  if (opts.pcts.some((p) => !(p >= 0 && p <= 1))) throw new Error('--frames must be fractions in [0, 1]');
  return opts;
}

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

function compile(source, fileName) {
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
    throw new Error(`syntax error: ${msg.join('; ')}`);
  }
  return out.outputText;
}

/** Shot order and declaration lines, read from the merged file the same way the
 * merger writes it: `const SHOTS ... = [Shot1_1, ...]` and `function ShotN_M(`. */
function shotTable(source) {
  const lines = source.split('\n');
  const m = /const\s+SHOTS\b[^=]*=\s*\[([^\]]*)\]/.exec(source);
  if (!m) return null;
  const names = m[1].split(',').map((s) => s.trim()).filter(Boolean);
  return names.map((name, index) => {
    const decl = new RegExp(`^\\s*(?:export\\s+)?(?:function\\s+${name}\\s*\\(|(?:const|let|var)\\s+${name}\\b)`);
    const at = lines.findIndex((l) => decl.test(l));
    const id = /^Shot(\d+)_(\d+)$/.exec(name);
    return {index, component: name, id: id ? `${id[1]}.${id[2]}` : null, line: at >= 0 ? at + 1 : null};
  });
}

// --- page ---------------------------------------------------------------------------

async function bundleHarness() {
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

async function resolveBrowser(explicit) {
  const path = explicit || process.env.LAYOUT_PROBE_BROWSER;
  if (path) return path;
  const {ensureBrowser} = await import('@remotion/renderer');
  const status = await ensureBrowser();
  if (!status.path) throw new Error(`no browser available: ${JSON.stringify(status)}`);
  return status.path;
}

async function openPage(browserPath, harnessJs) {
  const browser = await chromium.launch({executablePath: browserPath, headless: true, args: ['--disable-dev-shm-usage']});
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
  return {browser, page, logs};
}

// --- prototype rules (T6b will own the real ones) --------------------------------------

const pctLabel = (p) => `${Math.round(p * 100)}%`;
const quote = (t) => `'${t.length > 40 ? t.slice(0, 39) + '…' : t}'`;

function intersect(a, b) {
  const w = Math.min(a.x + a.w, b.x + b.w) - Math.max(a.x, b.x);
  const h = Math.min(a.y + a.h, b.y + b.h) - Math.max(a.y, b.y);
  return w > 1 && h > 1 ? {w: Math.round(w), h: Math.round(h)} : null;
}

function describe(e) {
  if (e.kind === 'text') return `chữ ${quote(e.text)}`;
  if (e.kind === 'kit') return `hình ${e.component}`;
  if (e.kind === 'svg') return 'khối <svg>';
  return `khối <${e.tag}>`;
}

function check(shots, band) {
  const found = new Map();
  const add = (shot, s, e, rule, severity, detail, extraKey = '') => {
    const key = `${shot.index}|${rule}|${e.line}|${e.kind}|${e.text ?? e.component ?? ''}|${extraKey}`;
    const prev = found.get(key);
    if (prev) {
      prev.frames.push(pctLabel(s.pct));
      return;
    }
    found.set(key, {shot: shot.id, shot_index: shot.index, line: shot.line, element_line: e.line, rule, severity, frames: [pctLabel(s.pct)], detail});
  };
  for (const shot of shots) {
    let largest = 0;
    for (const s of shot.samples) {
      if (s.error) {
        // A throw at some frame fails the real render too; tsc cannot see it.
        const first = s.error.split('\n')[0].slice(0, 200);
        add(shot, s, {line: null, kind: 'error', text: first}, 'runtime_error', 'blocking', `lỗi khi chạy shot: ${first}`);
      }
      if (s.pending_delay_render) {
        // Measured anyway, but something (font, Lottie JSON) had not loaded yet.
        add(shot, s, {line: null, kind: 'pending'}, 'not_settled', 'warning', `còn ${s.pending_delay_render} delayRender() chưa xong sau 5 giây — số đo có thể thiếu vật`);
      }
      const els = s.elements;
      els.forEach((e) => {
        const box = e.kind === 'text' ? e.text_rect : e.rect;
        if (!e.full_frame && e.kind !== 'text') largest = Math.max(largest, e.rect.w / 1920, e.rect.h / 1080);
        if (!e.full_frame) {
          const out = [];
          if (box.x < SAFE.left - 0.5) out.push(`trái x=${Math.round(box.x)} < ${SAFE.left}`);
          if (box.y < SAFE.top - 0.5) out.push(`trên y=${Math.round(box.y)} < ${SAFE.top}`);
          if (box.x + box.w > SAFE.right + 0.5) out.push(`phải x=${Math.round(box.x + box.w)} > ${SAFE.right}`);
          if (box.y + box.h > SAFE.bottom + 0.5) out.push(`dưới y=${Math.round(box.y + box.h)} > ${SAFE.bottom}`);
          if (out.length) add(shot, s, e, 'safe_area', 'blocking', `${describe(e)} ra ngoài vùng an toàn (${out.join(', ')})`);
          if (band) {
            const hit = band.edge === 'top' ? box.y < band.px : box.y + box.h > 1080 - band.px;
            if (hit) add(shot, s, e, 'subtitle_zone', 'blocking', `${describe(e)} lấn vùng phụ đề ${band.edge === 'top' ? `y < ${band.px}` : `y > ${1080 - band.px}`}`);
          }
        }
        if (e.kind !== 'text') return;
        if (e.client_width > 0 && e.scroll_width > e.client_width + 1) {
          add(shot, s, e, 'text_overflow', 'blocking', `nhãn ${quote(e.text)} tràn khung chữ (rộng ${e.scroll_width}px > width ${e.client_width}px)`);
        } else if (e.client_height > 0 && e.scroll_height > e.client_height + e.font_size / 2) {
          // Half a line, not 1 px: a line-height below the font's own ascent +
          // descent (lineHeight 1.2 with tall Vietnamese diacritics) already
          // makes scrollHeight a few px larger than an auto height.
          add(shot, s, e, 'text_overflow', 'blocking', `nhãn ${quote(e.text)} tràn khung chữ theo chiều cao (${e.scroll_height}px > height ${e.client_height}px)`);
        }
        if (e.font_size_rendered < MIN_FONT) {
          add(shot, s, e, 'min_font', 'warning', `chữ ${quote(e.text)} cỡ ${e.font_size_rendered}px < ${MIN_FONT}px`);
        }
      });
      const texts = els.map((e, i) => ({e, i})).filter(({e}) => e.kind === 'text');
      for (let a = 0; a < texts.length; a++) {
        for (let b = a + 1; b < texts.length; b++) {
          const A = texts[a], B = texts[b];
          if (B.e.text_parent === A.i || A.e.text_parent === B.i) continue; // nested text, one block
          const hit = intersect(A.e.text_rect, B.e.text_rect);
          if (hit) {
            add(shot, s, A.e, 'text_overlap', 'blocking',
              `nhãn ${quote(A.e.text)} (dòng ${A.e.line}) đè lên nhãn ${quote(B.e.text)} (dòng ${B.e.line}) — giao nhau ${hit.w}×${hit.h}px`, `${B.e.line}|${B.e.text}`);
          }
        }
      }
    }
    if (shot.samples.some((s) => s.elements.length) && largest < MIN_HERO_FRACTION) {
      found.set(`${shot.index}|hero`, {shot: shot.id, shot_index: shot.index, line: shot.line, element_line: null, rule: 'hero_size', severity: 'warning',
        frames: shot.samples.map((s) => pctLabel(s.pct)), detail: `vật lớn nhất chỉ chiếm ${Math.round(largest * 100)}% chiều khung (< ${MIN_HERO_FRACTION * 100}%)`});
    }
  }
  return [...found.values()].map((v) => ({...v, message: `Shot ${v.shot ?? v.shot_index + 1}, frame ${v.frames.join('/')}: ${v.detail}`}));
}

// --- main ----------------------------------------------------------------------------

async function probe(page, logs, path, opts) {
  const t = {};
  let t0 = performance.now();
  const source = readFileSync(path, 'utf-8');
  const js = compile(source, basename(path));
  t.compile = Math.round(performance.now() - t0);
  const table = shotTable(source);

  t0 = performance.now();
  logs.length = 0;
  const inputProps = opts.font ? {videoFont: opts.font} : {};
  const info = await page.evaluate(([code, props]) => window.cfProbe.load(code, props), [js, inputProps]);
  t.load = Math.round(performance.now() - t0);

  const count = table ? table.length : info.narrations;
  if (!count) throw new Error('cannot tell how many shots the script has (no SHOTS array, no narrations)');
  t0 = performance.now();
  const measured = await page.evaluate((o) => window.cfProbe.measure(o), {shots: count, duration: opts.duration, pcts: opts.pcts, settleTimeoutMs: 5000});
  t.measure = Math.round(performance.now() - t0);

  const shots = measured.map((m) => ({...(table?.[m.index] ?? {index: m.index, component: null, id: null, line: null}), samples: m.samples}));
  const sampleMs = shots.flatMap((s) => s.samples.map((x) => x.ms));
  const out = {
    version: 1,
    script: path,
    composition: {id: info.id, width: info.width, height: info.height, fps: info.fps},
    narrations: info.narrations,
    nominal_duration: opts.duration,
    frames: opts.pcts.map((p) => ({pct: p, frame: Math.round(p * (opts.duration - 1))})),
    shots,
    page_log: [...logs],
    timing_ms: {...t, samples: sampleMs.length, sample_avg: Math.round((sampleMs.reduce((a, b) => a + b, 0) / (sampleMs.length || 1)) * 10) / 10, sample_max: Math.max(0, ...sampleMs)},
  };
  if (opts.check) out.violations = check(shots, opts.band);
  return out;
}

async function main() {
  const opts = parseArgs(process.argv.slice(2));
  const tStart = performance.now();
  let t0 = performance.now();
  const harnessJs = await bundleHarness();
  const tBundle = Math.round(performance.now() - t0);

  t0 = performance.now();
  const browserPath = await resolveBrowser(opts.browser);
  const {browser, page, logs} = await openPage(browserPath, harnessJs);
  const tBrowser = Math.round(performance.now() - t0);
  const font = await page.evaluate((f) => window.cfProbe.fontStatus(f), opts.font || 'Be Vietnam Pro');

  const results = [];
  try {
    for (const script of opts.scripts) {
      t0 = performance.now();
      const r = await probe(page, logs, script, opts);
      r.timing_ms.script_total = Math.round(performance.now() - t0);
      r.font = font;
      r.browser = {path: browserPath, version: browser.version()};
      r.timing_ms.startup = {bundle_harness: tBundle, launch_browser_and_page: tBrowser};
      results.push(r);
      const v = r.violations ? `, ${r.violations.length} violations` : '';
      process.stderr.write(`${script}: ${r.shots.length} shots × ${opts.pcts.length} frames in ${r.timing_ms.script_total} ms${v}\n`);
    }
  } finally {
    await browser.close();
  }
  process.stderr.write(`startup: bundle ${tBundle} ms, browser+page ${tBrowser} ms; total wall ${Math.round(performance.now() - tStart)} ms\n`);
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
