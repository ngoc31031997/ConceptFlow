/**
 * conceptflow-mini/backdrops — whole-frame places a shot happens in, drawn in
 * the four depth layers of a Scene:
 *
 *   <Scene duration={duration} backdrop={MeadowBackdrop} camera={...}>...</Scene>
 *
 * draws `layer="sky"`, `"far"`, `"mid"` and `"near"` of the backdrop into the
 * matching layers, so the far hills drift slower than the figures and the
 * near grass is blurred in front of them.
 *
 * Every backdrop lays itself out from the frame's own size (useVideoConfig),
 * so it fits a landscape and a portrait video, and draws over the layer box
 * (useLayerBox) so a drifting layer never shows its edge. The middle of the
 * frame stays open for the shot's figures; colours are softer and darker than
 * a figure's so the figures stand out. Colour props take PALETTE values.
 */
import React from 'react';
import {random, useCurrentFrame, useVideoConfig} from 'remotion';
import {shadeOf, useSvgId} from './illustration';
import {useLayerBox} from './scene';

/** Which depth layer of a Scene a backdrop draws. */
export type BackdropLayer = 'sky' | 'far' | 'mid' | 'near';

/** What every backdrop takes. */
export interface BackdropProps {
  layer: BackdropLayer;
  /** Main colour of the place (sky, wall, water, body tissue). */
  color?: string;
  /** Colour of the ground, floor or the things standing on it. */
  ground?: string;
  /** Colour of the light or of the accents (sun, lamp, glowing cells). */
  light?: string;
  /** Turn off the backdrop's own idle motion (swaying, drifting). */
  still?: boolean;
}

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

/** A vertical gradient filling the layer box: the sky or the back wall. */
function SkyFill({top, bottom}: {top: string; bottom: string}) {
  const box = useLayerBox();
  const id = useSvgId('sky');
  return (
    <>
      <defs>
        <linearGradient id={id} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor={top} />
          <stop offset="1" stopColor={bottom} />
        </linearGradient>
      </defs>
      <rect x={box.left} y={box.top} width={box.width} height={box.height} fill={`url(#${id})`} />
    </>
  );
}

/** A soft round halo of light. */
function Halo({cx, cy, r, color, opacity = 0.7}: {cx: number; cy: number; r: number; color: string; opacity?: number}) {
  const id = useSvgId('halo');
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

function useSway(still: boolean | undefined, speed: number, phase = 0): number {
  const frame = useCurrentFrame();
  return still ? 0 : Math.sin(frame / speed + phase);
}

/** Outdoors on a sunny day: sky and sun, rolling hills, trees at the sides, grass in front. */
export function MeadowBackdrop({layer, color = '#8FD3FF', ground = '#4CAF50', light = '#FFD23F', still}: BackdropProps) {
  const {width: W, height: H} = useVideoConfig();
  const box = useLayerBox();
  const sway = useSway(still, 22);
  const gy = H * 0.7;
  if (layer === 'sky') {
    return (
      <LayerSvg>
        <SkyFill top={shadeOf(color, -0.1)} bottom={shadeOf(color, 0.45)} />
        <Halo cx={W * 0.78} cy={H * 0.2} r={Math.min(W, H) * 0.32} color={light} />
        <circle cx={W * 0.78} cy={H * 0.2} r={Math.min(W, H) * 0.07} fill={light} />
      </LayerSvg>
    );
  }
  if (layer === 'far') {
    return (
      <LayerSvg>
        <path d={`M ${box.left} ${gy - H * 0.08} Q ${W * 0.25} ${gy - H * 0.3} ${W * 0.5} ${gy - H * 0.1} T ${box.left + box.width} ${gy - H * 0.14} L ${box.left + box.width} ${box.top + box.height} L ${box.left} ${box.top + box.height} Z`} fill={shadeOf(ground, 0.35)} />
        <path d={`M ${box.left} ${gy - H * 0.02} Q ${W * 0.6} ${gy - H * 0.22} ${box.left + box.width} ${gy - H * 0.04} L ${box.left + box.width} ${box.top + box.height} L ${box.left} ${box.top + box.height} Z`} fill={shadeOf(ground, 0.18)} />
      </LayerSvg>
    );
  }
  if (layer === 'mid') {
    const tree = (x: number, s: number, k: number) => (
      <g key={k} transform={`translate(${x} ${gy}) rotate(${sway * 1.2})`}>
        <rect x={-s * 0.06} y={-s * 0.55} width={s * 0.12} height={s * 0.55} rx={s * 0.04} fill={shadeOf('#8A5A3C', -0.1)} />
        <circle cx={0} cy={-s * 0.75} r={s * 0.38} fill={shadeOf(ground, -0.08)} />
        <circle cx={s * 0.12} cy={-s * 0.82} r={s * 0.24} fill={shadeOf(ground, 0.12)} />
      </g>
    );
    return (
      <LayerSvg>
        <rect x={box.left} y={gy} width={box.width} height={box.top + box.height - gy} fill={ground} />
        <rect x={box.left} y={gy} width={box.width} height={H * 0.02} fill={shadeOf(ground, 0.2)} />
        {tree(W * 0.08, H * 0.55, 0)}
        {tree(W * 0.94, H * 0.48, 1)}
      </LayerSvg>
    );
  }
  const blade = (x: number, h: number, lean: number, k: number) => (
    <path key={k} d={`M ${x - 18} ${H + 40} Q ${x + lean + sway * 10} ${H - h * 0.5} ${x + lean * 1.4 + sway * 16} ${H - h} Q ${x + lean * 0.4} ${H - h * 0.4} ${x + 18} ${H + 40} Z`} fill={shadeOf(ground, -0.25)} />
  );
  return (
    <LayerSvg>
      {[0.02, 0.06, 0.1, 0.9, 0.95, 0.99].map((f, i) => blade(W * f, H * (0.22 + (i % 3) * 0.06), i < 3 ? 40 : -40, i))}
    </LayerSvg>
  );
}

const WHITE_FRAME = '#FFFFFF';

/** Indoors: a warm wall with a sunny window, a shelf at the side, a table edge in front. */
export function RoomBackdrop({layer, color = '#FFC857', ground = '#C9709A', light = '#FFF4C2'}: BackdropProps) {
  const {width: W, height: H} = useVideoConfig();
  const box = useLayerBox();
  const fy = H * 0.76;
  if (layer === 'sky') {
    return (
      <LayerSvg>
        <SkyFill top={shadeOf(color, -0.18)} bottom={shadeOf(color, 0.05)} />
        <rect x={box.left} y={fy} width={box.width} height={box.top + box.height - fy} fill={shadeOf(ground, -0.25)} />
      </LayerSvg>
    );
  }
  if (layer === 'far') {
    const w = Math.min(W, H) * 0.34;
    return (
      <LayerSvg>
        <Halo cx={W * 0.72} cy={H * 0.32} r={w * 1.3} color={light} opacity={0.55} />
        <rect x={W * 0.72 - w / 2} y={H * 0.32 - w * 0.6} width={w} height={w * 1.2} rx={18} fill={WHITE_FRAME} />
        <rect x={W * 0.72 - w / 2 + 16} y={H * 0.32 - w * 0.6 + 16} width={w - 32} height={w * 1.2 - 32} rx={10} fill={light} />
        <rect x={W * 0.72 - 6} y={H * 0.32 - w * 0.6 + 16} width={12} height={w * 1.2 - 32} fill={WHITE_FRAME} />
      </LayerSvg>
    );
  }
  if (layer === 'mid') {
    const sx = W * 0.06;
    const sw = W * 0.16;
    return (
      <LayerSvg>
        <rect x={sx} y={H * 0.3} width={sw} height={fy - H * 0.3} rx={14} fill={shadeOf(ground, 0.1)} />
        {[0.42, 0.56].map((f, i) => (
          <rect key={i} x={sx + 12} y={H * f} width={sw - 24} height={10} rx={5} fill={shadeOf(ground, -0.2)} />
        ))}
        <rect x={sx + sw * 0.2} y={H * 0.34} width={sw * 0.14} height={H * 0.08} rx={6} fill="#3D7BFF" />
        <rect x={sx + sw * 0.4} y={H * 0.36} width={sw * 0.12} height={H * 0.06} rx={6} fill="#E8453C" />
      </LayerSvg>
    );
  }
  return (
    <LayerSvg>
      <rect x={box.left} y={H * 0.9} width={box.width} height={box.top + box.height - H * 0.9} rx={30} fill={shadeOf(ground, -0.35)} />
    </LayerSvg>
  );
}
/** A town street: buildings with lit windows, a pavement with lamp posts, bushes in front. */
export function StreetBackdrop({layer, color = '#2D5BFF', ground = '#8F9BB3', light = '#FFD23F'}: BackdropProps) {
  const {width: W, height: H} = useVideoConfig();
  const box = useLayerBox();
  const gy = H * 0.74;
  if (layer === 'sky') {
    return (
      <LayerSvg>
        <SkyFill top={shadeOf(color, -0.35)} bottom={shadeOf(color, 0.3)} />
      </LayerSvg>
    );
  }
  if (layer === 'far') {
    const n = 7;
    const bw = box.width / n;
    return (
      <LayerSvg>
        {Array.from({length: n}, (_, i) => {
          const h = H * (0.3 + random(`street-${i}`) * 0.25);
          const x = box.left + i * bw;
          return (
            <g key={i}>
              <rect x={x + 6} y={gy - h} width={bw - 12} height={h} rx={10} fill={shadeOf(color, -0.45 + (i % 2) * 0.08)} />
              {[0.2, 0.45, 0.7].map((f, j) => (
                <rect key={j} x={x + bw * 0.3} y={gy - h + h * f} width={bw * 0.4} height={h * 0.1} rx={6} fill={light} opacity={random(`win-${i}-${j}`) > 0.4 ? 0.85 : 0.2} />
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
        <rect x={x - 6} y={gy - H * 0.38} width={12} height={H * 0.38} rx={6} fill={shadeOf(ground, -0.5)} />
        <Halo cx={x} cy={gy - H * 0.4} r={H * 0.12} color={light} opacity={0.6} />
        <circle cx={x} cy={gy - H * 0.4} r={H * 0.025} fill={light} />
      </g>
    );
    return (
      <LayerSvg>
        <rect x={box.left} y={gy} width={box.width} height={box.top + box.height - gy} fill={ground} />
        <rect x={box.left} y={gy} width={box.width} height={H * 0.015} fill={shadeOf(ground, 0.25)} />
        {lamp(W * 0.12, 0)}
        {lamp(W * 0.88, 1)}
      </LayerSvg>
    );
  }
  return (
    <LayerSvg>
      <ellipse cx={W * 0.02} cy={H * 1.0} rx={W * 0.12} ry={H * 0.14} fill={shadeOf('#2BB673', -0.3)} />
      <ellipse cx={W * 0.98} cy={H * 1.02} rx={W * 0.14} ry={H * 0.15} fill={shadeOf('#2BB673', -0.35)} />
    </LayerSvg>
  );
}

/** Inside the body: warm tissue lit from within, floating cells, vessel walls, big cells up close. */
export function InsideBodyBackdrop({layer, color = '#E8453C', ground = '#5B2C6F', light = '#FF9F43', still}: BackdropProps) {
  const {width: W, height: H} = useVideoConfig();
  const box = useLayerBox();
  const drift = useSway(still, 40);
  const id = useSvgId('tissue');
  if (layer === 'sky') {
    return (
      <LayerSvg>
        <defs>
          <radialGradient id={id} cx="0.5" cy="0.5" r="0.75">
            <stop offset="0" stopColor={shadeOf(light, 0.1)} />
            <stop offset="0.55" stopColor={color} />
            <stop offset="1" stopColor={shadeOf(ground, -0.3)} />
          </radialGradient>
        </defs>
        <rect x={box.left} y={box.top} width={box.width} height={box.height} fill={`url(#${id})`} />
      </LayerSvg>
    );
  }
  if (layer === 'far') {
    return (
      <LayerSvg>
        {Array.from({length: 9}, (_, i) => {
          const x = box.left + random(`cell-x-${i}`) * box.width;
          const y = box.top + random(`cell-y-${i}`) * box.height + drift * 12;
          const r = Math.min(W, H) * (0.03 + random(`cell-r-${i}`) * 0.05);
          return <circle key={i} cx={x} cy={y} r={r} fill={shadeOf(color, 0.25)} opacity={0.35} />;
        })}
      </LayerSvg>
    );
  }
  if (layer === 'mid') {
    return (
      <LayerSvg>
        <path d={`M ${box.left} ${box.top} L ${box.left + box.width} ${box.top} L ${box.left + box.width} ${H * 0.12} Q ${W * 0.5} ${H * 0.24} ${box.left} ${H * 0.1} Z`} fill={shadeOf(ground, 0.1)} />
        <path d={`M ${box.left} ${box.top + box.height} L ${box.left + box.width} ${box.top + box.height} L ${box.left + box.width} ${H * 0.88} Q ${W * 0.5} ${H * 0.76} ${box.left} ${H * 0.9} Z`} fill={shadeOf(ground, 0.1)} />
      </LayerSvg>
    );
  }
  const cell = (cx: number, cy: number, r: number, k: number) => (
    <g key={k}>
      <circle cx={cx} cy={cy + drift * 8} r={r} fill={shadeOf(color, 0.12)} />
      <circle cx={cx - r * 0.25} cy={cy - r * 0.25 + drift * 8} r={r * 0.35} fill={shadeOf(color, -0.25)} />
    </g>
  );
  return (
    <LayerSvg>
      {cell(W * 0.0, H * 0.05, Math.min(W, H) * 0.2, 0)}
      {cell(W * 1.0, H * 0.95, Math.min(W, H) * 0.24, 1)}
    </LayerSvg>
  );
}

/** Under water: light from the surface, distant rocks, swaying seaweed, bubbles up close. */
export function UnderwaterBackdrop({layer, color = '#2BB6A8', ground = '#9A6A45', light = '#8FD3FF', still}: BackdropProps) {
  const {width: W, height: H} = useVideoConfig();
  const box = useLayerBox();
  const frame = useCurrentFrame();
  const sway = useSway(still, 18);
  if (layer === 'sky') {
    return (
      <LayerSvg>
        <SkyFill top={shadeOf(light, 0.2)} bottom={shadeOf(color, -0.55)} />
        {[0.2, 0.45, 0.7].map((f, i) => (
          <path key={i} d={`M ${W * f} ${box.top} L ${W * f + W * 0.06} ${box.top} L ${W * f + W * 0.18} ${H} L ${W * f + W * 0.04} ${H} Z`} fill="#FFFFFF" opacity={0.08} />
        ))}
      </LayerSvg>
    );
  }
  if (layer === 'far') {
    return (
      <LayerSvg>
        <path d={`M ${box.left} ${H * 0.86} Q ${W * 0.3} ${H * 0.7} ${W * 0.55} ${H * 0.84} T ${box.left + box.width} ${H * 0.8} L ${box.left + box.width} ${box.top + box.height} L ${box.left} ${box.top + box.height} Z`} fill={shadeOf(color, -0.4)} />
      </LayerSvg>
    );
  }
  if (layer === 'mid') {
    const weed = (x: number, h: number, k: number) => (
      <path key={k} d={`M ${x} ${H} C ${x + 40 + sway * 30} ${H - h * 0.4}, ${x - 40 - sway * 30} ${H - h * 0.7}, ${x + sway * 40} ${H - h}`} stroke={shadeOf('#2BB673', -0.1)} strokeWidth={22} strokeLinecap="round" fill="none" />
    );
    return (
      <LayerSvg>
        <rect x={box.left} y={H * 0.9} width={box.width} height={box.top + box.height - H * 0.9} fill={ground} />
        {[0.05, 0.1, 0.88, 0.94].map((f, i) => weed(W * f, H * (0.35 + (i % 2) * 0.12), i))}
      </LayerSvg>
    );
  }
  return (
    <LayerSvg>
      {Array.from({length: 6}, (_, i) => {
        const side = i % 2 === 0 ? 0.04 : 0.96;
        const rise = still ? 0 : ((frame * 2 + i * 120) % (H + 200));
        return <circle key={i} cx={W * side + (random(`bub-${i}`) - 0.5) * W * 0.06} cy={H + 100 - rise} r={10 + random(`bubr-${i}`) * 20} fill="#FFFFFF" opacity={0.45} />;
      })}
    </LayerSvg>
  );
}

/** Space: a deep sky full of stars, a ringed planet, rocks drifting close by. */
export function SpaceBackdrop({layer, color = '#140B3A', ground = '#7B3FC4', light = '#FFC857', still}: BackdropProps) {
  const {width: W, height: H} = useVideoConfig();
  const box = useLayerBox();
  const frame = useCurrentFrame();
  if (layer === 'sky') {
    return (
      <LayerSvg>
        <SkyFill top={color} bottom={shadeOf(ground, -0.55)} />
      </LayerSvg>
    );
  }
  if (layer === 'far') {
    return (
      <LayerSvg>
        {Array.from({length: 70}, (_, i) => {
          const twinkle = still ? 1 : 0.6 + 0.4 * Math.sin(frame / 10 + i);
          return (
            <circle key={i} cx={box.left + random(`star-x-${i}`) * box.width} cy={box.top + random(`star-y-${i}`) * box.height} r={2 + random(`star-r-${i}`) * 3} fill="#FFFFFF" opacity={0.8 * twinkle} />
          );
        })}
      </LayerSvg>
    );
  }
  if (layer === 'mid') {
    const r = Math.min(W, H) * 0.12;
    return (
      <LayerSvg>
        <Halo cx={W * 0.82} cy={H * 0.24} r={r * 2} color={light} opacity={0.35} />
        <circle cx={W * 0.82} cy={H * 0.24} r={r} fill={light} />
        <circle cx={W * 0.82 + r * 0.3} cy={H * 0.24 + r * 0.2} r={r * 0.8} fill={shadeOf(light, -0.15)} opacity={0.5} />
        <ellipse cx={W * 0.82} cy={H * 0.24} rx={r * 1.8} ry={r * 0.35} fill="none" stroke={shadeOf(ground, 0.3)} strokeWidth={10} />
      </LayerSvg>
    );
  }
  const rock = (cx: number, cy: number, s: number, k: number) => (
    <g key={k} transform={`rotate(${still ? 0 : frame * 0.2} ${cx} ${cy})`}>
      <ellipse cx={cx} cy={cy} rx={s} ry={s * 0.8} fill={shadeOf(ground, -0.35)} />
      <circle cx={cx - s * 0.3} cy={cy - s * 0.2} r={s * 0.18} fill={shadeOf(ground, -0.55)} />
    </g>
  );
  return (
    <LayerSvg>
      {rock(W * 0.03, H * 0.9, Math.min(W, H) * 0.12, 0)}
      {rock(W * 0.97, H * 0.08, Math.min(W, H) * 0.09, 1)}
    </LayerSvg>
  );
}
