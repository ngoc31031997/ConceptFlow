/**
 * Page side of the layout probe (CR-048 T6a spike — not wired into any pipeline).
 *
 * layout_probe.mjs bundles this file once with esbuild, opens it in headless
 * Chromium and, per merged script:
 *   1. hands in the script already compiled to CommonJS (every JSX element
 *      carries a `data-cf-line` attribute = its line in the merged file);
 *   2. `load()` evaluates it with a `require` that only knows the modules a
 *      generated script may import, and captures the <Composition> passed to
 *      registerRoot;
 *   3. `measure()` draws the composition with @remotion/player's <Thumbnail>
 *      at shot i's frame f (segments of a nominal length, so exactly one shot
 *      is mounted), waits until every delayRender() handle is released (font,
 *      Lottie JSON) and reads the DOM boxes of what the script drew.
 *
 * Nothing here screenshots: a sample costs one React commit plus layout.
 */
import React from 'react';
import * as ReactNS from 'react';
import * as JSXRuntime from 'react/jsx-runtime';
import {flushSync} from 'react-dom';
import {createRoot, type Root} from 'react-dom/client';
import {Thumbnail} from '@remotion/player';
import * as Remotion from 'remotion';
import * as Segments from '../conceptflow-mini/segments';
import * as Primitives from '../conceptflow-mini/primitives';
import * as Kit from '../conceptflow-mini/illustration';
import * as Lottie from '../conceptflow-mini/lottie';

type Rect = {x: number; y: number; w: number; h: number};

type Box = {
  kind: 'text' | 'svg' | 'kit' | 'shape';
  /** Line of the JSX element in the merged script that produced this box. */
  line: number | null;
  rect: Rect;
  opacity: number;
  full_frame?: boolean;
  component?: string;
  tag?: string;
  // text only
  text?: string;
  text_rect?: Rect;
  lines?: number;
  font_size?: number;
  font_size_rendered?: number;
  scroll_width?: number;
  client_width?: number;
  scroll_height?: number;
  client_height?: number;
  /** Index (in this sample's list) of the nearest enclosing text element. */
  text_parent?: number | null;
};

type Sample = {
  frame: number;
  pct: number;
  global_frame: number;
  elements: Box[];
  pending_delay_render?: number;
  error?: string;
  ms: number;
};

// --- script loading ---------------------------------------------------------------

// Kit parts that render inside an <svg> (a <g>, an <ellipse>): wrapping them in
// a <div> would break the drawing, so they stay unlabelled.
const SVG_PARTS = new Set(['Face', 'GroundShadow']);

/**
 * A kit component wrapped so its box can be found and named: a
 * `display: contents` div generates no box of its own, so layout is unchanged,
 * and it carries the component name and the line of the element that used it.
 */
function tagged(name: string, Comp: React.ComponentType<Record<string, unknown>>) {
  const Tagged = (props: Record<string, unknown>) => {
    const {'data-cf-line': line, ...rest} = props;
    return (
      <div data-cf-kit={name} data-cf-line={line as number | undefined} style={{display: 'contents'}}>
        <Comp {...rest} />
      </div>
    );
  };
  Tagged.displayName = `probe(${name})`;
  return Tagged;
}

function tagModule(mod: Record<string, unknown>): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const [name, value] of Object.entries(mod)) {
    const isComponent = typeof value === 'function' && /^[A-Z]/.test(name) && !SVG_PARTS.has(name);
    out[name] = isComponent ? tagged(name, value as React.ComponentType<Record<string, unknown>>) : value;
  }
  return out;
}

const esm = (ns: object, def?: unknown) => ({...ns, ...(def === undefined ? {} : {default: def}), __esModule: true});

let registered: (() => React.ReactElement) | null = null;

const MODULES: Record<string, unknown> = {
  react: esm(ReactNS, React),
  'react/jsx-runtime': JSXRuntime,
  remotion: esm({
    ...Remotion,
    registerRoot: (fn: () => React.ReactElement) => {
      registered = fn;
    },
  }),
  './conceptflow-mini/segments': esm(Segments),
  './conceptflow-mini/primitives': esm(Primitives),
  './conceptflow-mini/illustration': esm(tagModule(Kit as unknown as Record<string, unknown>)),
  './conceptflow-mini/lottie': esm(tagModule(Lottie as unknown as Record<string, unknown>)),
};

type Loaded = {
  component: React.ComponentType<Record<string, unknown>>;
  width: number;
  height: number;
  fps: number;
  id: string;
  narrations: number | null;
};

let loaded: Loaded | null = null;

function findComposition(node: unknown): React.ReactElement | null {
  if (!node || typeof node !== 'object') return null;
  const el = node as React.ReactElement<Record<string, unknown>>;
  if (el.type === Remotion.Composition) return el;
  const children = el.props?.children;
  for (const child of Array.isArray(children) ? children : [children]) {
    const found = findComposition(child);
    if (found) return found;
  }
  return null;
}

function load(js: string, inputProps: Record<string, unknown>) {
  (window as unknown as {cfProbeInputProps: unknown}).cfProbeInputProps = inputProps;
  registered = null;
  loaded = null;
  const module = {exports: {} as Record<string, unknown>};
  const require = (id: string) => {
    if (!(id in MODULES)) throw new Error(`import '${id}' is not available to a generated script`);
    return MODULES[id];
  };
  // eslint-disable-next-line no-new-func
  new Function('require', 'module', 'exports', js)(require, module, module.exports);
  if (!registered) throw new Error('the script never called registerRoot()');
  const comp = findComposition((registered as () => React.ReactElement)());
  if (!comp) throw new Error('registerRoot() did not render a <Composition>');
  const p = comp.props as Record<string, unknown>;
  const narrations = module.exports.narrations;
  loaded = {
    component: p.component as React.ComponentType<Record<string, unknown>>,
    width: Number(p.width),
    height: Number(p.height),
    fps: Number(p.fps),
    id: String(p.id),
    narrations: Array.isArray(narrations) ? narrations.length : null,
  };
  const {component, ...info} = loaded;
  return info;
}

// --- measuring ---------------------------------------------------------------------

const round = (v: number) => Math.round(v * 10) / 10;

function rectOf(r: DOMRect, origin: DOMRect, scale: number): Rect {
  return {
    x: round((r.left - origin.left) / scale),
    y: round((r.top - origin.top) / scale),
    w: round(r.width / scale),
    h: round(r.height / scale),
  };
}

function union(rects: DOMRect[]): DOMRect | null {
  const real = rects.filter((r) => r.width > 0 || r.height > 0);
  if (!real.length) return null;
  const l = Math.min(...real.map((r) => r.left));
  const t = Math.min(...real.map((r) => r.top));
  const rr = Math.max(...real.map((r) => r.right));
  const b = Math.max(...real.map((r) => r.bottom));
  return new DOMRect(l, t, rr - l, b - t);
}

function alpha(color: string): number {
  if (!color || color === 'transparent') return 0;
  const m = /rgba?\(([^)]+)\)/.exec(color);
  if (!m) return 1;
  const parts = m[1].split(/[\s,/]+/).filter(Boolean);
  return parts.length >= 4 ? Number(parts[3]) : 1;
}

function isPainted(cs: CSSStyleDeclaration, tag: string): boolean {
  if (['img', 'canvas', 'video'].includes(tag)) return true;
  if (alpha(cs.backgroundColor) > 0 || cs.backgroundImage !== 'none') return true;
  return ['Top', 'Right', 'Bottom', 'Left'].some(
    (s) =>
      parseFloat(cs.getPropertyValue(`border-${s.toLowerCase()}-width`)) > 0 &&
      cs.getPropertyValue(`border-${s.toLowerCase()}-style`) !== 'none' &&
      alpha(cs.getPropertyValue(`border-${s.toLowerCase()}-color`)) > 0,
  );
}

function ownText(el: Element): string {
  let s = '';
  for (const n of Array.from(el.childNodes)) {
    if (n.nodeType === Node.TEXT_NODE) s += n.textContent ?? '';
  }
  return s.replace(/\s+/g, ' ').trim();
}

function collect(stage: HTMLElement, width: number, height: number): Box[] {
  const origin = stage.getBoundingClientRect();
  const scale = origin.width / width || 1;
  const out: Box[] = [];
  const index = new Map<Element, number>();
  const opacityOf = new Map<Element, number>();
  const effectiveOpacity = (el: Element): number => {
    if (el === stage || !el) return 1;
    const cached = opacityOf.get(el);
    if (cached !== undefined) return cached;
    const own = parseFloat(getComputedStyle(el).opacity);
    const v = (Number.isFinite(own) ? own : 1) * (el.parentElement ? effectiveOpacity(el.parentElement) : 1);
    opacityOf.set(el, v);
    return v;
  };

  for (const el of Array.from(stage.querySelectorAll('*'))) {
    const owner = el.closest('[data-cf-line]');
    if (!owner) continue; // Thumbnail / Stage scaffolding, not drawn by the script
    const tag = el.tagName.toLowerCase();
    const cs = getComputedStyle(el);
    if (cs.display === 'none' || cs.visibility === 'hidden') continue;
    const opacity = effectiveOpacity(el);
    if (opacity < 0.01) continue;
    const lineAttr = owner.getAttribute('data-cf-line');
    const line = lineAttr ? Number(lineAttr) : null;
    const kitRoot = el.closest('[data-cf-kit]');
    const inSvg = el instanceof SVGElement && el.ownerSVGElement !== null;

    let entry: Box | null = null;
    if (el.hasAttribute('data-cf-kit')) {
      const box = union(Array.from(el.children).map((c) => c.getBoundingClientRect()));
      if (!box) continue;
      entry = {kind: 'kit', component: el.getAttribute('data-cf-kit') ?? '', line, rect: rectOf(box, origin, scale), opacity};
    } else if (inSvg) {
      if (tag !== 'text') continue;
      const text = (el.textContent ?? '').replace(/\s+/g, ' ').trim();
      if (!text) continue;
      const r = el.getBoundingClientRect();
      const ctm = (el as SVGGraphicsElement).getScreenCTM();
      const fs = parseFloat(cs.fontSize);
      entry = {
        kind: 'text', tag, line, text: text.slice(0, 80), rect: rectOf(r, origin, scale), text_rect: rectOf(r, origin, scale),
        opacity, font_size: fs, font_size_rendered: round((fs * (ctm ? Math.hypot(ctm.a, ctm.b) : 1)) / scale), lines: 1,
      };
    } else if (tag === 'svg') {
      if (kitRoot) continue; // counted as the kit component's box
      entry = {kind: 'svg', tag, line, rect: rectOf(el.getBoundingClientRect(), origin, scale), opacity};
    } else {
      const text = ownText(el);
      const r = el.getBoundingClientRect();
      if (text) {
        const range = document.createRange();
        range.selectNodeContents(el);
        const tr = range.getBoundingClientRect();
        const tops = new Set(Array.from(range.getClientRects()).filter((c) => c.width > 0).map((c) => Math.round(c.top)));
        const fs = parseFloat(cs.fontSize);
        const html = el as HTMLElement;
        const layoutScale = html.offsetWidth > 0 ? r.width / html.offsetWidth : 1;
        const inline = cs.display === 'inline';
        entry = {
          kind: 'text', tag, line, text: text.slice(0, 80), rect: rectOf(r, origin, scale), text_rect: rectOf(tr, origin, scale),
          opacity, lines: tops.size, font_size: fs, font_size_rendered: round((fs * layoutScale) / scale),
          ...(inline
            ? {}
            : {scroll_width: html.scrollWidth, client_width: html.clientWidth, scroll_height: html.scrollHeight, client_height: html.clientHeight}),
        };
      } else if (!kitRoot && isPainted(cs, tag)) {
        entry = {kind: 'shape', tag, line, rect: rectOf(r, origin, scale), opacity};
      }
    }
    if (!entry || (entry.rect.w <= 0 && entry.rect.h <= 0)) continue;
    const {x, y, w, h} = entry.rect;
    if (x <= 0.5 && y <= 0.5 && x + w >= width - 0.5 && y + h >= height - 0.5) entry.full_frame = true;
    if (entry.kind === 'text') {
      let p = el.parentElement;
      entry.text_parent = null;
      while (p && p !== stage) {
        if (index.has(p) && out[index.get(p)!].kind === 'text') {
          entry.text_parent = index.get(p)!;
          break;
        }
        p = p.parentElement;
      }
    }
    index.set(el, out.length);
    out.push(entry);
  }
  return out;
}

const macrotask = () => new Promise<void>((r) => setTimeout(r, 0));
// Handles that already outlived one settle timeout (a clip whose JSON never
// loads): they would otherwise hold every later sample for the full timeout.
const stale = new Set<unknown>();
const held = () => ((window as unknown as {remotion_delayRenderHandles?: unknown[]}).remotion_delayRenderHandles ?? []).filter((h) => !stale.has(h));

/**
 * Waits until nothing holds a delayRender() handle (the project font, a Lottie
 * clip's JSON) for two macrotasks in a row, so the React updates those
 * releases schedule have been committed too. Returns how many handles were
 * still held when `timeoutMs` ran out (0 = settled); those are then ignored.
 */
async function settle(timeoutMs: number): Promise<number> {
  const t0 = performance.now();
  let quiet = 0;
  while (quiet < 2) {
    await macrotask();
    quiet = held().length === 0 ? quiet + 1 : 0;
    if (performance.now() - t0 > timeoutMs) {
      const stuck = held();
      stuck.forEach((h) => stale.add(h));
      return stuck.length;
    }
  }
  return 0;
}

let root: Root | null = null;

async function measure(opts: {shots: number; duration: number; pcts: number[]; settleTimeoutMs: number}) {
  if (!loaded) throw new Error('load() a script first');
  const {component, width, height, fps} = loaded;
  const {shots, duration, pcts, settleTimeoutMs} = opts;
  const segments = Array.from({length: shots}, (_, i) => ({startFrame: i * duration, durationInFrames: duration}));
  const inputProps = {segments};
  const stage = document.getElementById('cf-stage') as HTMLElement;
  stage.style.width = `${width}px`;
  stage.style.height = `${height}px`;
  root?.unmount();
  root = createRoot(stage);
  let caught: Error | null = null;
  let epoch = 0;
  const draw = (frame: number) =>
    flushSync(() =>
      root!.render(
        <Thumbnail
          key={epoch}
          component={component}
          inputProps={inputProps}
          compositionWidth={width}
          compositionHeight={height}
          fps={fps}
          durationInFrames={shots * duration}
          frameToDisplay={frame}
          overflowVisible
          style={{width, height}}
          errorFallback={({error}) => {
            caught = error;
            return null;
          }}
        />,
      ),
    );

  const result: {index: number; samples: Sample[]}[] = [];
  for (let i = 0; i < shots; i++) {
    const samples: Sample[] = [];
    for (const pct of pcts) {
      const t0 = performance.now();
      const frame = Math.round(pct * (duration - 1));
      const global = i * duration + frame;
      caught = null;
      let error: string | undefined;
      try {
        draw(global);
      } catch (err) {
        error = String((err as Error)?.stack ?? err);
      }
      const stuck = await settle(settleTimeoutMs);
      if (caught) error = String((caught as Error).stack ?? caught);
      const sample: Sample = {frame, pct, global_frame: global, elements: [], ms: 0};
      if (error) {
        sample.error = error.slice(0, 2000);
        epoch++; // an error boundary stays tripped until the Thumbnail remounts
      } else {
        sample.elements = collect(stage, width, height);
      }
      if (stuck) sample.pending_delay_render = stuck;
      sample.ms = round(performance.now() - t0);
      samples.push(sample);
    }
    result.push({index: i, samples});
  }
  return result;
}

/**
 * Whether Chromium really draws text in `family`. document.fonts only lists
 * web fonts (@font-face), never a font installed in the system, so it cannot
 * answer this for the rendering image's fonts; instead, a sample string is
 * measured with two different fallbacks — if the family exists, both widths
 * are the family's own and agree.
 */
function fontStatus(family: string) {
  const ctx = document.createElement('canvas').getContext('2d')!;
  const sample = 'Vì sao răng bị sâu? WMmwil 0123';
  const width = (font: string) => {
    ctx.font = font;
    return ctx.measureText(sample).width;
  };
  const available = [400, 700].every(
    (w) => width(`${w} 36px '${family}', monospace`) === width(`${w} 36px '${family}', serif`),
  );
  return {family, available};
}

(window as unknown as {cfProbe: unknown}).cfProbe = {load, measure, fontStatus};
