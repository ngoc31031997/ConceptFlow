/**
 * conceptflow-mini/backdrops — whole-frame places a shot happens in, drawn in
 * the four depth layers of a Scene:
 *
 *   <Scene duration={duration} backdrop={MeadowBackdrop} backdropProps={{color, ground, light}} camera={...}>...</Scene>
 *
 * draws `layer="sky"`, `"far"`, `"mid"` and `"near"` of the backdrop into the
 * matching layers, so the far ridges drift slower than the figures and the
 * near plants are blurred in front of them.
 *
 * Backdrops are built from the procedural generators (noise-driven ridges,
 * canopies, grass, stars, cells...), WebGL shader layers and lit `Shape`s.
 * They take every colour from props and never choose one: `color` is the main
 * colour of the place (sky, wall, water, tissue), `ground` the ground or the
 * things standing on it, `light` the colour of the light or accents. All three
 * must be PALETTE values. Every backdrop lays itself out from the frame's own
 * size (useVideoConfig) over the layer box (useLayerBox) so it fits a landscape
 * and a portrait video and a drifting layer never shows its edge. The middle of
 * the frame stays open for the shot's figures.
 */
import React from 'react';
import {useVideoConfig} from 'remotion';
import {mix, withAlpha} from './film';
import {Canopy, Cells, Clouds, Coral, Dunes, Grass, Hills, Particles, Ridges, Rocks, Stars, Waves} from './procedural';
import {useLayerBox} from './scene';
import {ShaderLayer} from './shader';
import {Shape} from './material';

/** Which depth layer of a Scene a backdrop draws. */
export type BackdropLayer = 'sky' | 'far' | 'mid' | 'near';

/** What every backdrop takes. */
export interface BackdropProps {
  layer: BackdropLayer;
  /** Main colour of the place (sky, wall, water, body tissue). */
  color: string;
  /** Colour of the ground, floor or the things standing on it. */
  ground: string;
  /** Colour of the light or of the accents (sun, lamp, glowing cells). */
  light: string;
  /** Turn off the backdrop's own idle motion (swaying, drifting). */
  still?: boolean;
}

const DARK = '#000000';
const LIGHT_MIX = '#FFFFFF';

/** An `<svg>` covering the layer box, in frame coordinates. */
function LayerSvg({children}: {children: React.ReactNode}) {
  const box = useLayerBox();
  return (
    <svg
      width={box.width}
      height={box.height}
      viewBox={`${box.left} ${box.top} ${box.width} ${box.height}`}
      style={{position: 'absolute', left: box.left, top: box.top, overflow: 'visible'}}
    >
      {children}
    </svg>
  );
}

/** A soft round halo of light. */
function Halo({cx, cy, r, color, opacity = 0.7}: {cx: number; cy: number; r: number; color: string; opacity?: number}) {
  const id = React.useId().replace(/[^a-zA-Z0-9_-]/g, '');
  return (
    <>
      <defs>
        <radialGradient id={id}>
          <stop offset="0" stopColor={color} stopOpacity={opacity} />
          <stop offset="1" stopColor={color} stopOpacity={0} />
        </radialGradient>
      </defs>
      <circle cx={cx} cy={cy} r={r} fill={`url(#${id})`} />
    </>
  );
}

/** Outdoors by day: shader sky, sun, noise ridges and hills, canopy trees, grass in front. */
export function MeadowBackdrop({layer, color, ground, light, still}: BackdropProps) {
  const {width: W, height: H} = useVideoConfig();
  const box = useLayerBox();
  const gy = H * 0.7;
  if (layer === 'sky') {
    return (
      <>
        <ShaderLayer shader="sky" colors={[mix(color, DARK, 0.1), mix(color, LIGHT_MIX, 0.45), mix(color, LIGHT_MIX, 0.8)]} intensity={0.9} />
        <LayerSvg>
          <Halo cx={W * 0.78} cy={H * 0.2} r={Math.min(W, H) * 0.32} color={light} />
          <Shape material="emissive" color={light} ellipse={{cx: W * 0.78, cy: H * 0.2, rx: Math.min(W, H) * 0.07, ry: Math.min(W, H) * 0.07}} />
          <Clouds y={H * 0.22} spread={H * 0.1} color={mix(color, LIGHT_MIX, 0.85)} count={4} />
        </LayerSvg>
      </>
    );
  }
  if (layer === 'far') {
    return (
      <LayerSvg>
        <Ridges y={gy - H * 0.14} amp={H * 0.16} color={mix(ground, mix(color, LIGHT_MIX, 0.3), 0.55)} seed="far-ridge" rough={0.9} />
        <Hills y={gy - H * 0.06} amp={H * 0.1} color={mix(ground, mix(color, LIGHT_MIX, 0.3), 0.3)} seed="far-hill" />
      </LayerSvg>
    );
  }
  if (layer === 'mid') {
    const tree = (x: number, s: number, k: number) => (
      <g key={k}>
        <Shape material="wood" color="#8A5A3C" rect={{x: x - s * 0.06, y: gy - s * 0.55, w: s * 0.12, h: s * 0.55}} />
        <Canopy cx={x} cy={gy - s * 0.78} r={s * 0.42} color={ground} seed={`tree-${k}`} />
      </g>
    );
    return (
      <LayerSvg>
        <rect x={box.left} y={gy} width={box.width} height={box.top + box.height - gy} fill={ground} />
        <rect x={box.left} y={gy} width={box.width} height={H * 0.02} fill={mix(ground, light, 0.25)} />
        {tree(W * 0.08, H * 0.55, 0)}
        {tree(W * 0.94, H * 0.48, 1)}
      </LayerSvg>
    );
  }
  return (
    <LayerSvg>
      <Grass baseY={H + 30} height={H * 0.3} color={mix(ground, DARK, 0.3)} count={46} seed="near-grass" still={still} />
    </LayerSvg>
  );
}

/** Indoors: a wall lit through a window, a shelf at the side, a table edge in front. */
export function RoomBackdrop({layer, color, ground, light, still}: BackdropProps) {
  const {width: W, height: H} = useVideoConfig();
  const box = useLayerBox();
  const fy = H * 0.76;
  if (layer === 'sky') {
    return (
      <LayerSvg>
        <rect x={box.left} y={box.top} width={box.width} height={fy - box.top} fill={mix(color, DARK, 0.12)} />
        <rect x={box.left} y={fy} width={box.width} height={box.top + box.height - fy} fill={mix(ground, DARK, 0.3)} />
      </LayerSvg>
    );
  }
  if (layer === 'far') {
    const w = Math.min(W, H) * 0.34;
    const wx = W * 0.72;
    const wy = H * 0.32;
    return (
      <>
        <ShaderLayer shader="godrays" colors={[light, light]} intensity={0.35} pos={[0.72, 0.32]} />
        <LayerSvg>
          <Halo cx={wx} cy={wy} r={w * 1.3} color={light} opacity={0.55} />
          <Shape material="matte" color={mix(color, LIGHT_MIX, 0.8)} rect={{x: wx - w / 2, y: wy - w * 0.6, w, h: w * 1.2}} />
          <Shape material="emissive" color={light} rect={{x: wx - w / 2 + 16, y: wy - w * 0.6 + 16, w: w - 32, h: w * 1.2 - 32}} />
          <rect x={wx - 6} y={wy - w * 0.6 + 16} width={12} height={w * 1.2 - 32} fill={mix(color, LIGHT_MIX, 0.8)} />
        </LayerSvg>
        <LayerSvg>
          <Particles count={26} color={withAlpha(light, 0.8)} size={3} seed="dust" still={still} speed={0.15} />
        </LayerSvg>
      </>
    );
  }
  if (layer === 'mid') {
    const sx = W * 0.06;
    const sw = W * 0.16;
    return (
      <LayerSvg>
        <Shape material="wood" color={mix(ground, LIGHT_MIX, 0.1)} rect={{x: sx, y: H * 0.3, w: sw, h: fy - H * 0.3}} />
        {[0.42, 0.56].map((f, i) => (
          <Shape key={i} material="wood" color={mix(ground, DARK, 0.25)} rect={{x: sx + 12, y: H * f, w: sw - 24, h: 10}} />
        ))}
        <Shape material="glossy" color={mix(color, ground, 0.5)} rect={{x: sx + sw * 0.2, y: H * 0.34, w: sw * 0.14, h: H * 0.08}} />
        <Shape material="glossy" color={mix(light, ground, 0.5)} rect={{x: sx + sw * 0.4, y: H * 0.36, w: sw * 0.12, h: H * 0.06}} />
      </LayerSvg>
    );
  }
  return (
    <LayerSvg>
      <Shape material="wood" color={mix(ground, DARK, 0.35)} rect={{x: box.left, y: H * 0.9, w: box.width, h: box.top + box.height - H * 0.9}} />
    </LayerSvg>
  );
}

/** A town street: buildings with lit windows, a pavement with lamp posts, bushes in front. */
export function StreetBackdrop({layer, color, ground, light}: BackdropProps) {
  const {width: W, height: H} = useVideoConfig();
  const box = useLayerBox();
  const gy = H * 0.74;
  if (layer === 'sky') {
    return <ShaderLayer shader="sky" colors={[mix(color, DARK, 0.35), mix(color, LIGHT_MIX, 0.3), mix(color, LIGHT_MIX, 0.5)]} intensity={0.5} />;
  }
  if (layer === 'far') {
    const n = 7;
    const bw = box.width / n;
    return (
      <LayerSvg>
        {Array.from({length: n}, (_, i) => {
          const h = H * (0.3 + 0.25 * (((i * 37) % 11) / 11));
          const x = box.left + i * bw;
          return (
            <g key={i}>
              <Shape material="stone" color={mix(color, DARK, 0.45 - (i % 2) * 0.08)} rect={{x: x + 6, y: gy - h, w: bw - 12, h}} />
              {[0.2, 0.45, 0.7].map((f, j) => (
                <Shape key={j} material="emissive" color={light} rect={{x: x + bw * 0.3, y: gy - h + h * f, w: bw * 0.4, h: h * 0.1}} opacity={(i * 3 + j) % 3 === 0 ? 0.2 : 0.85} />
              ))}
            </g>
          );
        })}
      </LayerSvg>
    );
  }
  if (layer === 'mid') {
    const lamp = (x: number, k: number) => (
      <g key={k}>
        <Shape material="metal" color={mix(ground, DARK, 0.5)} rect={{x: x - 6, y: gy - H * 0.38, w: 12, h: H * 0.38}} />
        <Halo cx={x} cy={gy - H * 0.4} r={H * 0.12} color={light} opacity={0.6} />
        <Shape material="emissive" color={light} ellipse={{cx: x, cy: gy - H * 0.4, rx: H * 0.025, ry: H * 0.025}} />
      </g>
    );
    return (
      <LayerSvg>
        <Shape material="stone" color={ground} rect={{x: box.left, y: gy, w: box.width, h: box.top + box.height - gy}} />
        <rect x={box.left} y={gy} width={box.width} height={H * 0.015} fill={mix(ground, LIGHT_MIX, 0.25)} />
        {lamp(W * 0.12, 0)}
        {lamp(W * 0.88, 1)}
      </LayerSvg>
    );
  }
  return (
    <LayerSvg>
      <Canopy cx={W * 0.02} cy={H * 1.02} r={H * 0.16} color={mix(ground, color, 0.3)} seed="bush-l" />
      <Canopy cx={W * 0.98} cy={H * 1.03} r={H * 0.18} color={mix(ground, color, 0.3)} seed="bush-r" />
    </LayerSvg>
  );
}

/** Inside the body: tissue lit from within, floating cells, vessel walls, big cells up close. */
export function InsideBodyBackdrop({layer, color, ground, light, still}: BackdropProps) {
  const {width: W, height: H} = useVideoConfig();
  const box = useLayerBox();
  if (layer === 'sky') {
    return <ShaderLayer shader="tissue" colors={[mix(light, color, 0.5), color, mix(ground, light, 0.4)]} intensity={0.9} />;
  }
  if (layer === 'far') {
    return (
      <>
        <LayerSvg>
          <Cells count={22} a={mix(color, LIGHT_MIX, 0.25)} b={mix(color, ground, 0.35)} wall={mix(color, LIGHT_MIX, 0.4)} nucleus={mix(ground, color, 0.3)} seed="far-tissue" opacity={0.3} />
        </LayerSvg>
        <ShaderLayer shader="fog" colors={[mix(color, LIGHT_MIX, 0.3), color]} intensity={0.35} />
      </>
    );
  }
  if (layer === 'mid') {
    return (
      <LayerSvg>
        <Shape
          material="organic"
          color={mix(ground, LIGHT_MIX, 0.1)}
          d={`M ${box.left} ${box.top} L ${box.left + box.width} ${box.top} L ${box.left + box.width} ${H * 0.12} Q ${W * 0.5} ${H * 0.24} ${box.left} ${H * 0.1} Z`}
        />
        <Shape
          material="organic"
          color={mix(ground, LIGHT_MIX, 0.1)}
          d={`M ${box.left} ${box.top + box.height} L ${box.left + box.width} ${box.top + box.height} L ${box.left + box.width} ${H * 0.88} Q ${W * 0.5} ${H * 0.76} ${box.left} ${H * 0.9} Z`}
        />
        <Particles count={30} color={withAlpha(light, 0.8)} size={4} seed="plasma" still={still} speed={0.25} />
      </LayerSvg>
    );
  }
  const cell = (cx: number, cy: number, r: number, k: number) => (
    <g key={k}>
      <Shape material="organic" color={mix(color, LIGHT_MIX, 0.12)} ellipse={{cx, cy, rx: r, ry: r}} />
      <Shape material="organic" color={mix(color, DARK, 0.25)} ellipse={{cx: cx - r * 0.25, cy: cy - r * 0.25, rx: r * 0.35, ry: r * 0.35}} />
    </g>
  );
  return (
    <LayerSvg>
      {cell(W * 0.0, H * 0.05, Math.min(W, H) * 0.2, 0)}
      {cell(W * 1.0, H * 0.95, Math.min(W, H) * 0.24, 1)}
    </LayerSvg>
  );
}

/** Under water: caustics from the surface, distant dunes, coral and seaweed, bubbles up close. */
export function UnderwaterBackdrop({layer, color, ground, light, still}: BackdropProps) {
  const {width: W, height: H} = useVideoConfig();
  const box = useLayerBox();
  if (layer === 'sky') {
    return (
      <>
        <ShaderLayer shader="water" colors={[mix(light, color, 0.3), mix(color, DARK, 0.55), mix(light, LIGHT_MIX, 0.4)]} intensity={0.7} />
        <ShaderLayer shader="godrays" colors={[mix(light, LIGHT_MIX, 0.3), light]} intensity={0.4} pos={[0.45, -0.05]} />
      </>
    );
  }
  if (layer === 'far') {
    return (
      <>
        <LayerSvg>
          <Dunes y={H * 0.82} amp={H * 0.1} color={mix(color, DARK, 0.4)} seed="sea-floor" />
          <Rocks baseY={H * 0.86} count={5} size={H * 0.05} color={mix(color, DARK, 0.5)} seed="far-rock" />
        </LayerSvg>
        <ShaderLayer shader="fog" colors={[mix(color, DARK, 0.2), color]} intensity={0.4} />
      </>
    );
  }
  if (layer === 'mid') {
    const weed = (x: number, h: number, k: number) => (
      <path
        key={k}
        d={`M ${x} ${H} C ${x + 40} ${H - h * 0.4}, ${x - 40} ${H - h * 0.7}, ${x} ${H - h}`}
        stroke={mix(ground, color, 0.4)}
        strokeWidth={22}
        strokeLinecap="round"
        fill="none"
      />
    );
    return (
      <LayerSvg>
        <Shape material="stone" color={ground} rect={{x: box.left, y: H * 0.9, w: box.width, h: box.top + box.height - H * 0.9}} />
        <Coral baseY={H * 0.92} count={3} height={H * 0.3} color={mix(light, ground, 0.5)} seed="coral" />
        <Waves y={H * 0.06} amp={H * 0.015} color={light} layers={2} still={still} />
        {[0.05, 0.1, 0.88, 0.94].map((f, i) => weed(W * f, H * (0.35 + (i % 2) * 0.12), i))}
      </LayerSvg>
    );
  }
  return (
    <LayerSvg>
      <Particles count={16} color={withAlpha(mix(light, LIGHT_MIX, 0.5), 0.9)} size={14} seed="bubbles" still={still} speed={1.2} />
    </LayerSvg>
  );
}

/** Space: a nebula sky full of stars, a ringed planet, rocks close by. */
export function SpaceBackdrop({layer, color, ground, light}: BackdropProps) {
  const {width: W, height: H} = useVideoConfig();
  const box = useLayerBox();
  if (layer === 'sky') {
    return <ShaderLayer shader="nebula" colors={[color, mix(ground, DARK, 0.5), mix(ground, light, 0.3)]} intensity={0.7} />;
  }
  if (layer === 'far') {
    return (
      <LayerSvg>
        <Stars count={90} color={mix(light, LIGHT_MIX, 0.8)} seed="far-stars" />
      </LayerSvg>
    );
  }
  if (layer === 'mid') {
    const r = Math.min(W, H) * 0.12;
    const cx = W * 0.82;
    const cy = H * 0.24;
    return (
      <LayerSvg>
        <Halo cx={cx} cy={cy} r={r * 2} color={light} opacity={0.35} />
        <Shape material="matte" color={light} ellipse={{cx, cy, rx: r, ry: r}} />
        <ellipse cx={cx} cy={cy} rx={r * 1.8} ry={r * 0.35} fill="none" stroke={mix(ground, LIGHT_MIX, 0.3)} strokeWidth={10} />
      </LayerSvg>
    );
  }
  return (
    <LayerSvg>
      <Rocks baseY={H * 0.97} count={2} size={Math.min(W, H) * 0.09} color={mix(ground, DARK, 0.35)} seed="near-rock" />
      <Shape material="stone" color={mix(ground, DARK, 0.4)} ellipse={{cx: box.left + box.width * 0.97, cy: H * 0.08, rx: Math.min(W, H) * 0.09, ry: Math.min(W, H) * 0.07}} />
    </LayerSvg>
  );
}
