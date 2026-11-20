/**
 * Preview host for one illustration of the library.
 *
 * Bundled ONCE by illustration_preview.mjs; each preview hands the figure in as
 * input props — already-compiled JavaScript for a library drawing, or the name
 * of a built-in kit component — so a preview costs a render, not a webpack build.
 */
import React from 'react';
import * as ReactNS from 'react';
import * as JSXRuntime from 'react/jsx-runtime';
import * as Remotion from 'remotion';
import {registerRoot, Composition, AbsoluteFill, getInputProps} from 'remotion';
import * as Kit from '../conceptflow-mini/illustration';
import * as SceneKit from '../conceptflow-mini/scene';
import * as Backdrops from '../conceptflow-mini/backdrops';
import * as FilmKit from '../conceptflow-mini/film';
import * as MaterialKit from '../conceptflow-mini/material';
import * as SkeletonKit from '../conceptflow-mini/skeleton';
import * as ProceduralKit from '../conceptflow-mini/procedural';
import * as ShaderKit from '../conceptflow-mini/shader';
import * as RigKit from '../conceptflow-mini/rig';
import * as KitsModule from '../conceptflow-mini/kits';
import {SampleCompositions} from '../gallery/language-sample';
import {LookProvider} from '../conceptflow-mini/film';
import {currentKit} from '../conceptflow-mini/kits';

export const PREVIEW_SIZE = 720;
/** A backdrop is previewed as a whole landscape frame, scaled down. */
const BACKDROP_PREVIEW = {width: 1280, height: 720};
const PREVIEW_FRAMES = 60;

type HostProps = {
  /** CommonJS output of esbuild for the asset's TSX; empty for a built-in. */
  js?: string;
  /** Exported component name to render. */
  name?: string;
  /** Extra props for the figure (mood, decay...). */
  props?: Record<string, unknown>;
  background?: string;
  /** "figure" (default) or "backdrop": a backdrop is drawn in a Scene's layers. */
  kind?: 'figure' | 'backdrop';
};

const MODULES: Record<string, unknown> = {
  react: ReactNS,
  'react/jsx-runtime': JSXRuntime,
  remotion: Remotion,
  './conceptflow-mini/illustration': Kit,
  './conceptflow-mini/scene': SceneKit,
  './conceptflow-mini/backdrops': Backdrops,
  './conceptflow-mini/film': FilmKit,
  './conceptflow-mini/material': MaterialKit,
  './conceptflow-mini/skeleton': SkeletonKit,
  './conceptflow-mini/procedural': ProceduralKit,
  './conceptflow-mini/shader': ShaderKit,
  './conceptflow-mini/rig': RigKit,
  './conceptflow-mini/kits': KitsModule,
};

function load(js: string): Record<string, unknown> {
  const module = {exports: {} as Record<string, unknown>};
  const require = (id: string) => {
    if (!(id in MODULES)) throw new Error(`import '${id}' is not allowed in a library drawing`);
    return MODULES[id];
  };
  // eslint-disable-next-line no-new-func
  new Function('require', 'module', 'exports', js)(require, module, module.exports);
  return module.exports;
}

function Preview() {
  const {js = '', name = '', props = {}, background = '#FFF4D6', kind = 'figure'} = getInputProps() as HostProps;
  const builtin = {...(Kit as unknown as Record<string, unknown>), ...(Backdrops as unknown as Record<string, unknown>)};
  const kit = currentKit();
  const exports = js ? load(js) : builtin;
  const Figure = exports[name] as React.ComponentType<Record<string, unknown>> | undefined;
  if (typeof Figure !== 'function') {
    throw new Error(`component '${name}' is not exported`);
  }
  if (kind === 'backdrop') {
    // A loaded drawing's props are unknown to the type system; the style check
    // already required it to take `layer`.
    const backdrop = Figure as React.ComponentType<SceneKit.BackdropInput>;
    const colors = props as Omit<SceneKit.BackdropInput, 'layer'>;
    return (
      <LookProvider look={kit.previewLook}>
        <SceneKit.Scene duration={PREVIEW_FRAMES} backdrop={backdrop} backdropProps={colors} />
      </LookProvider>
    );
  }
  return (
    <LookProvider look={kit.previewLook}>
      <AbsoluteFill style={{backgroundColor: background}}>
        <Figure x={PREVIEW_SIZE / 2} y={PREVIEW_SIZE / 2} size={PREVIEW_SIZE * 0.8} {...props} />
      </AbsoluteFill>
    </LookProvider>
  );
}

export const FRAME_SIZE = {width: 1280, height: 720};
/** Height of the bottom band that holds the on-screen text lines. */
const TEXT_BAND = 150;

/** Closed vocabulary of a storyboard frame: cell centre as a share of the drawing area (the frame above the text band). */
export const POSITIONS = {
  upper_left: [1 / 6, 1 / 6], upper_center: [1 / 2, 1 / 6], upper_right: [5 / 6, 1 / 6],
  center_left: [1 / 6, 1 / 2], center: [1 / 2, 1 / 2], center_right: [5 / 6, 1 / 2],
  lower_left: [1 / 6, 5 / 6], lower_center: [1 / 2, 5 / 6], lower_right: [5 / 6, 5 / 6],
  split_left: [1 / 4, 1 / 2], split_right: [3 / 4, 1 / 2], full: [1 / 2, 1 / 2],
} as const;
export type FramePosition = keyof typeof POSITIONS;

/**
 * Largest share of the drawing area (the frame above the text band) a drawing's box may fill, by camera
 * view and tier (hero = the subject of the shot).
 */
export const CAMERA_SCALE = {
  wide: {hero: 0.8, support: 0.55},
  medium: {hero: 0.9, support: 0.65},
  close: {hero: 1.0, support: 0.75},
  comparison: {hero: 0.85, support: 0.85},
} as const;
export type CameraView = keyof typeof CAMERA_SCALE;

/** Share of the area's width one drawing may take, so two side-by-side drawings never touch. */
const WIDTH_SHARE = 0.42;
/** Clear space kept between a drawing and the edges of the drawing area. */
const EDGE_MARGIN = 24;
/** Size a drawing is rendered at while its box is measured; the box scales linearly with `size`. */
const MEASURE_SIZE = 100;
const MEASURE_ANCHOR = 500;

type Placement = {
  js?: string; name: string; props?: Record<string, unknown>;
  position: FramePosition; tier?: string;
};
type FrameProps = {
  kit?: unknown;
  background: {color: string; backdrop?: {js: string; name: string; props: Omit<SceneKit.BackdropInput, 'layer'>} | null};
  placements: Placement[];
  camera_view: CameraView;
  on_screen_text: {text: string; min_px: number}[];
  text_color: string;
};
/** A drawing's painted box per unit of `size`: its width, height and centre offset from the anchor point (x, y). */
type UnitBox = {w: number; h: number; dx: number; dy: number};

/**
 * Union of the figure boxes (the `svg` elements a drawing renders, whose size follows `size`) under `root`, per unit
 * of size, relative to the anchor at (MEASURE_ANCHOR, MEASURE_ANCHOR). */
function measureBox(root: HTMLElement, name: string): UnitBox {
  const origin = root.getBoundingClientRect();
  let left = Infinity; let top = Infinity; let right = -Infinity; let bottom = -Infinity;
  root.querySelectorAll('svg').forEach((el) => {
    const r = el.getBoundingClientRect();
    if (r.width <= 0 || r.height <= 0) return;
    left = Math.min(left, r.left - origin.left); top = Math.min(top, r.top - origin.top);
    right = Math.max(right, r.right - origin.left); bottom = Math.max(bottom, r.bottom - origin.top);
  });
  if (!Number.isFinite(left)) throw new Error(`component '${name}' renders no svg figure: its box cannot be measured`);
  return {
    w: (right - left) / MEASURE_SIZE, h: (bottom - top) / MEASURE_SIZE,
    dx: ((left + right) / 2 - MEASURE_ANCHOR) / MEASURE_SIZE, dy: ((top + bottom) / 2 - MEASURE_ANCHOR) / MEASURE_SIZE,
  };
}

/**
 * One storyboard frame: several drawings on a 3x3 grid (or split / full) plus text lines in the bottom band.
 * Each drawing is first rendered unseen to measure its real painted box (a Person is taller than wide, a Germ
 * is square), then sized to fit its cell by that box and kept inside the drawing area, so nothing is cut and
 * nothing reaches the text band.
 */
function StoryboardFrame() {
  const {background, placements, camera_view, on_screen_text, text_color} = getInputProps() as unknown as FrameProps;
  const kit = currentKit();
  const area = {w: FRAME_SIZE.width, h: FRAME_SIZE.height - TEXT_BAND};
  const view = CAMERA_SCALE[camera_view];
  if (!view) throw new Error(`camera_view '${camera_view}' is not in the closed vocabulary`);
  const figures = placements.map((pl) => {
    if (!POSITIONS[pl.position]) throw new Error(`position '${pl.position}' is not in the closed vocabulary`);
    const Figure = (pl.js ? load(pl.js) : (Kit as unknown as Record<string, unknown>))[pl.name] as
      React.ComponentType<Record<string, unknown>> | undefined;
    if (typeof Figure !== 'function') throw new Error(`component '${pl.name}' is not exported`);
    return Figure;
  });
  const probes = React.useRef<(HTMLDivElement | null)[]>([]);
  const [boxes, setBoxes] = React.useState<UnitBox[] | null>(null);
  const [handle] = React.useState(() => Remotion.delayRender('measure storyboard drawings'));
  React.useLayoutEffect(() => {
    setBoxes(placements.map((pl, i) => measureBox(probes.current[i] as HTMLDivElement, pl.name)));
  }, []);
  React.useEffect(() => {
    if (boxes) Remotion.continueRender(handle);
  }, [boxes]);

  const drawn = boxes ? placements.map((pl, i) => {
    const Figure = figures[i];
    const box = boxes[i];
    const [cellX, cellY] = POSITIONS[pl.position];
    const share = view[pl.tier === 'hero' ? 'hero' : 'support'];
    const full = pl.position === 'full';
    const fitW = (area.w * (full ? 1 : WIDTH_SHARE) * share) / box.w;
    const fitH = (area.h * share) / box.h;
    const edgeW = (area.w - 2 * EDGE_MARGIN) / box.w;
    const edgeH = (area.h - 2 * EDGE_MARGIN) / box.h;
    const size = Math.min(fitW, fitH, edgeW, edgeH);
    const halfW = (box.w * size) / 2;
    const halfH = (box.h * size) / 2;
    const cx = Math.min(Math.max(cellX * area.w, EDGE_MARGIN + halfW), area.w - EDGE_MARGIN - halfW);
    const cy = Math.min(Math.max(cellY * area.h, EDGE_MARGIN + halfH), area.h - EDGE_MARGIN - halfH);
    return <Figure key={i} x={cx - box.dx * size} y={cy - box.dy * size} size={size} {...(pl.props ?? {})} />;
  }) : null;
  const text = (
    <div style={{position: 'absolute', left: 0, right: 0, bottom: 0, height: TEXT_BAND, display: 'flex', flexDirection: 'column',
      justifyContent: 'center', alignItems: 'center', gap: 8, color: text_color, fontFamily: 'sans-serif', fontWeight: 700,
      textAlign: 'center', textShadow: '0 2px 6px rgba(0,0,0,0.45)'}}>
      {on_screen_text.map((line, i) => <div key={i} style={{fontSize: line.min_px}}>{line.text}</div>)}
    </div>
  );
  const backdrop = background.backdrop;
  const Backdrop = backdrop ? ((load(backdrop.js)[backdrop.name]) as React.ComponentType<SceneKit.BackdropInput> | undefined) : undefined;
  if (backdrop && typeof Backdrop !== 'function') throw new Error(`component '${backdrop.name}' is not exported`);
  const probe = (
    <div style={{position: 'absolute', left: 0, top: 0, width: 0, height: 0, visibility: 'hidden'}}>
      {placements.map((pl, i) => {
        const Figure = figures[i];
        return (
          <div key={i} ref={(el) => { probes.current[i] = el; }} style={{position: 'absolute', left: 0, top: 0}}>
            <Figure x={MEASURE_ANCHOR} y={MEASURE_ANCHOR} size={MEASURE_SIZE} {...(pl.props ?? {})} />
          </div>
        );
      })}
    </div>
  );
  return (
    <LookProvider look={kit.previewLook}>
      <AbsoluteFill style={{backgroundColor: background.color}}>
        {Backdrop && backdrop ? (
          <SceneKit.Scene duration={PREVIEW_FRAMES} backdrop={Backdrop} backdropProps={backdrop.props}>{drawn}</SceneKit.Scene>
        ) : drawn}
        {text}
        {boxes ? null : probe}
      </AbsoluteFill>
    </LookProvider>
  );
}

registerRoot(() => (
  <>
  <Composition id="frame" component={StoryboardFrame} width={FRAME_SIZE.width} height={FRAME_SIZE.height} fps={30}
    durationInFrames={PREVIEW_FRAMES} />
  <SampleCompositions />
  <Composition
    id="asset"
    component={Preview}
    width={PREVIEW_SIZE}
    height={PREVIEW_SIZE}
    fps={30}
    durationInFrames={PREVIEW_FRAMES}
    calculateMetadata={({props}) => {
      const {kind} = props as HostProps;
      return kind === 'backdrop' ? BACKDROP_PREVIEW : {width: PREVIEW_SIZE, height: PREVIEW_SIZE};
    }}
  />
  </>
));
