/**
 * conceptflow-mini/scene — what turns a shot from "a few figures on a flat
 * colour" into a place the camera moves through:
 *
 *   - `Scene` stacks a shot in depth layers (sky, far, mid, the shot's own
 *     figures, near) that fill the whole frame and a little beyond it, drifts
 *     each layer at its own speed when the camera moves (parallax), blurs the
 *     near layer like a shallow depth of field, and darkens the corners;
 *   - `Camera` is the same camera move for a shot that has no Scene;
 *   - `Glow`, `LightRays` and `Vignette` are the light: a halo behind what
 *     matters, rays from a source, darker corners;
 *   - `KeywordText` is the one kind of on-screen text a shot keeps: a short
 *     keyword that pops in on a coloured block.
 *
 * Coordinates are the frame's own (0..width, 0..height of the composition),
 * so the same code fits a landscape and a portrait video. Layers and lights
 * carry `data-cf-layer`: the layout check reads them as scenery, not as
 * objects that must stay inside the safe area. The figures' layer and
 * `Camera` carry `data-cf-camera="moved"` while the camera is off its rest
 * point: an object it pushes past the safe area is the shot's framing, and
 * the layout check only warns about it unless it is the shot's main object.
 */
import React from 'react';
import {AbsoluteFill, Easing, interpolate, spring, useCurrentFrame, useVideoConfig} from 'remotion';

/** Where the camera looks: the frame point shown at the centre, and the zoom. */
export type CameraPoint = {x: number; y: number; zoom?: number};

/** A camera move across a shot, from one point to another. */
export type CameraMove = {
  from: CameraPoint;
  to?: CameraPoint;
  /** Fraction of the shot's duration at which the move starts. */
  start?: number;
  /** Fraction of the shot's duration at which the move ends. */
  end?: number;
};

type CameraState = {x: number; y: number; zoom: number};

/** How far each layer reaches past the frame edge, as a share of the frame. */
const OVERSCAN = 0.12;
/** How much of the camera's move each depth layer follows (1 = the figures). */
const DEPTH = {sky: 0, far: 0.3, mid: 0.7, near: 1.35} as const;
const SCENERY = 'scene';
const LIGHT = 'light';
const CAMERA_MOVED = 'moved';
/** Half a pixel: closer than this to the rest point counts as not moved. */
const CAMERA_REST_TOLERANCE = 0.5;
const clamp = {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'} as const;

/**
 * The camera at the current frame of a shot `duration` frames long. Zoom
 * never goes below 1, so the frame never shows past the scene's edge.
 */
export function useCamera(move: CameraMove | undefined, duration: number): CameraState {
  const frame = useCurrentFrame();
  const {width, height} = useVideoConfig();
  const centre = {x: width / 2, y: height / 2, zoom: 1};
  if (!move) return centre;
  const from = {...centre, ...move.from};
  const to = {...from, ...(move.to ?? move.from)};
  const a = Math.max(0, duration * (move.start ?? 0));
  const b = Math.max(a + 1, duration * (move.end ?? 0.85));
  const t = interpolate(frame, [a, b], [0, 1], {...clamp, easing: Easing.inOut(Easing.cubic)});
  return {
    x: from.x + (to.x - from.x) * t,
    y: from.y + (to.y - from.y) * t,
    zoom: Math.max(1, (from.zoom ?? 1) + ((to.zoom ?? 1) - (from.zoom ?? 1)) * t),
  };
}

/**
 * `data-cf-camera` for the figures seen through `cam`: set while the camera
 * is zoomed in or looks away from the frame centre, else absent.
 */
function cameraMark(cam: CameraState, width: number, height: number): string | undefined {
  const moved =
    cam.zoom - 1 > 1e-3 ||
    Math.abs(cam.x - width / 2) > CAMERA_REST_TOLERANCE ||
    Math.abs(cam.y - height / 2) > CAMERA_REST_TOLERANCE;
  return moved ? CAMERA_MOVED : undefined;
}

/**
 * The CSS transform that shows a layer of depth `depth` through the camera:
 * `depth` 1 follows the camera exactly, 0 stays still, more than 1 moves
 * faster than the figures (closer to the lens).
 */
function lensTransform(cam: CameraState, depth: number, width: number, height: number): string {
  const zoom = 1 + (cam.zoom - 1) * depth;
  const x = width / 2 + (cam.x - width / 2) * depth;
  const y = height / 2 + (cam.y - height / 2) * depth;
  return `translate(${width / 2 - x * zoom}px, ${height / 2 - y * zoom}px) scale(${zoom})`;
}

/**
 * A camera move for a shot without a Scene: pushes in, pulls out or pans by
 * moving everything inside it. `duration` is the shot's own length in frames.
 */
export function Camera({
  duration,
  from,
  to,
  start,
  end,
  children,
}: CameraMove & {duration: number; children: React.ReactNode}) {
  const {width, height} = useVideoConfig();
  const cam = useCamera({from, to, start, end}, duration);
  return (
    <AbsoluteFill
      data-cf-camera={cameraMark(cam, width, height)}
      style={{transformOrigin: '0 0', transform: lensTransform(cam, 1, width, height)}}
    >
      {children}
    </AbsoluteFill>
  );
}

/** The box a depth layer is drawn in: the frame plus its overscan, in frame px. */
export type LayerBox = {left: number; top: number; width: number; height: number};

const LayerContext = React.createContext<LayerBox | null>(null);

/**
 * The box of the depth layer this component is drawn in. A backdrop draws
 * its `<svg viewBox>` over this box so it still covers the frame while the
 * layer drifts. Outside a Scene it is the frame itself.
 */
export function useLayerBox(): LayerBox {
  const box = React.useContext(LayerContext);
  const {width, height} = useVideoConfig();
  return box ?? {left: 0, top: 0, width, height};
}

function Layer({
  depth,
  cam,
  blur,
  overscan,
  children,
}: {
  depth: number;
  cam: CameraState;
  blur?: number;
  overscan: boolean;
  children: React.ReactNode;
}) {
  const {width, height} = useVideoConfig();
  const pad = overscan ? OVERSCAN : 0;
  const box = {left: -width * pad, top: -height * pad, width: width * (1 + 2 * pad), height: height * (1 + 2 * pad)};
  return (
    <AbsoluteFill
      data-cf-layer={overscan ? SCENERY : undefined}
      data-cf-camera={overscan ? undefined : cameraMark(cam, width, height)}
      style={{transformOrigin: '0 0', transform: lensTransform(cam, depth, width, height), filter: blur ? `blur(${blur}px)` : undefined}}
    >
      <LayerContext.Provider value={box}>
        <div style={{position: 'absolute', left: box.left, top: box.top, width: box.width, height: box.height}}>
          {/* Children keep frame coordinates: shift back by the overscan. */}
          <div style={{position: 'absolute', left: -box.left, top: -box.top, width, height}}>{children}</div>
        </div>
      </LayerContext.Provider>
    </AbsoluteFill>
  );
}

/**
 * A shot as a place: depth layers that fill the frame, the shot's figures
 * between the mid and near layers, and an optional camera move that drifts
 * each layer at its own speed.
 *
 * - `sky`: the farthest layer, never moves (a Backdrop, a backdrop drawing's
 *   sky layer).
 * - `far`, `mid`: scenery behind the figures (hills, walls, shelves).
 * - `near`: scenery in front of the figures, blurred by `blurNear` px (leaves,
 *   a table edge, a cell wall seen up close). Keep it at the frame edges.
 * - `backdrop`: a backdrop drawing (MeadowBackdrop, a library backdrop...);
 *   it fills every layer not given explicitly, with `backdropProps`.
 * - `camera`: the move, in frame coordinates; zoom ≥ 1. A pan at zoom 1 stays
 *   within about 10% of the frame, or the overscan runs out.
 * - `vignette`: how dark the corners get, 0 to 0.6.
 * - `duration`: the shot's own length in frames.
 */
export function Scene({
  duration,
  backdrop: Backdrop,
  backdropProps,
  sky,
  far,
  mid,
  near,
  camera,
  vignette = 0.35,
  blurNear = 10,
  children,
}: {
  duration: number;
  backdrop?: React.ComponentType<{layer: 'sky' | 'far' | 'mid' | 'near'} & Record<string, unknown>>;
  backdropProps?: Record<string, unknown>;
  sky?: React.ReactNode;
  far?: React.ReactNode;
  mid?: React.ReactNode;
  near?: React.ReactNode;
  camera?: CameraMove;
  vignette?: number;
  blurNear?: number;
  children?: React.ReactNode;
}) {
  const cam = useCamera(camera, duration);
  const fromBackdrop = (layer: 'sky' | 'far' | 'mid' | 'near') =>
    Backdrop ? <Backdrop layer={layer} {...backdropProps} /> : null;
  sky = sky ?? fromBackdrop('sky');
  far = far ?? fromBackdrop('far');
  mid = mid ?? fromBackdrop('mid');
  near = near ?? fromBackdrop('near');
  return (
    <AbsoluteFill style={{overflow: 'hidden'}}>
      {sky ? <Layer depth={DEPTH.sky} cam={cam} overscan>{sky}</Layer> : null}
      {far ? <Layer depth={DEPTH.far} cam={cam} overscan>{far}</Layer> : null}
      {mid ? <Layer depth={DEPTH.mid} cam={cam} overscan>{mid}</Layer> : null}
      <Layer depth={1} cam={cam} overscan={false}>{children}</Layer>
      {near ? <Layer depth={DEPTH.near} cam={cam} blur={blurNear} overscan>{near}</Layer> : null}
      {vignette > 0 ? <Vignette strength={vignette} /> : null}
    </AbsoluteFill>
  );
}

/**
 * Darker corners that pull the eye to the centre. Stays fixed to the screen
 * whatever the camera does.
 */
export function Vignette({strength = 0.35, color = '#000000'}: {strength?: number; color?: string}) {
  const alpha = Math.round(Math.max(0, Math.min(0.6, strength)) * 255)
    .toString(16)
    .padStart(2, '0');
  return (
    <AbsoluteFill
      data-cf-layer={LIGHT}
      style={{pointerEvents: 'none', backgroundImage: `radial-gradient(ellipse at center, ${color}00 55%, ${color}${alpha} 100%)`}}
    />
  );
}

/**
 * A soft halo of light centred on (`x`, `y`), radius `r`: the warm glow
 * inside a fig, the light around an idea. `pulse` breathes it slowly.
 */
export function Glow({
  x,
  y,
  r,
  color = '#FFD23F',
  opacity = 0.6,
  pulse = false,
}: {
  x: number;
  y: number;
  r: number;
  color?: string;
  opacity?: number;
  pulse?: boolean;
}) {
  const frame = useCurrentFrame();
  const breathe = pulse ? 1 + Math.sin(frame / 18) * 0.06 : 1;
  const radius = r * breathe;
  return (
    <div
      data-cf-layer={LIGHT}
      style={{
        position: 'absolute',
        left: x - radius,
        top: y - radius,
        width: radius * 2,
        height: radius * 2,
        borderRadius: '50%',
        opacity,
        backgroundImage: `radial-gradient(circle, ${color} 0%, ${color}66 40%, ${color}00 70%)`,
        pointerEvents: 'none',
      }}
    />
  );
}

/**
 * Rays of light fanning out from (`x`, `y`): sunlight through leaves, a lamp,
 * a revelation. They turn slowly unless `still`.
 */
export function LightRays({
  x,
  y,
  count = 9,
  length,
  color = '#FFFFFF',
  opacity = 0.18,
  still = false,
}: {
  x: number;
  y: number;
  count?: number;
  length?: number;
  color?: string;
  opacity?: number;
  still?: boolean;
}) {
  const frame = useCurrentFrame();
  const {width, height} = useVideoConfig();
  const reach = length ?? Math.hypot(width, height);
  const turn = still ? 0 : frame * 0.08;
  const spread = 360 / count;
  const half = spread * 0.22;
  const rays = Array.from({length: count}, (_, i) => {
    const a = ((i * spread + turn) * Math.PI) / 180;
    const b = ((i * spread + turn + 2 * half) * Math.PI) / 180;
    return `M ${x} ${y} L ${x + Math.cos(a) * reach} ${y + Math.sin(a) * reach} L ${x + Math.cos(b) * reach} ${y + Math.sin(b) * reach} Z`;
  });
  return (
    <AbsoluteFill data-cf-layer={LIGHT} style={{pointerEvents: 'none'}}>
      <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} style={{position: 'absolute', inset: 0}}>
        {rays.map((d, i) => (
          <path key={i} d={d} fill={color} opacity={opacity} />
        ))}
      </svg>
    </AbsoluteFill>
  );
}

/**
 * A short keyword (one to three words) that pops in on a coloured block,
 * slightly tilted: the label of the concept being said, never the narration.
 * `x`, `y` are the block's centre, `width` its fixed width (the text wraps
 * inside it), `start` the fraction of `duration` at which it appears.
 */
export function KeywordText({
  children,
  x,
  y,
  width,
  duration,
  size = 88,
  color = '#FFFFFF',
  background = '#2D5BFF',
  start = 0,
  tilt = -3,
}: {
  children: React.ReactNode;
  x: number;
  y: number;
  width: number;
  duration: number;
  size?: number;
  color?: string;
  background?: string;
  start?: number;
  tilt?: number;
}) {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const pop = spring({frame: frame - duration * start, fps, config: {damping: 12, stiffness: 160}});
  const lineHeight = 1.15;
  return (
    <div
      style={{
        position: 'absolute',
        left: x - width / 2,
        top: y - (size * lineHeight + size * 0.5) / 2,
        width,
        padding: `${size * 0.25}px ${size * 0.35}px`,
        boxSizing: 'border-box',
        backgroundColor: background,
        borderRadius: size * 0.18,
        color,
        fontSize: size,
        fontWeight: 700,
        lineHeight,
        textAlign: 'center',
        overflowWrap: 'break-word',
        opacity: Math.min(1, pop * 1.5),
        transform: `rotate(${tilt}deg) scale(${pop})`,
      }}
    >
      {children}
    </div>
  );
}
