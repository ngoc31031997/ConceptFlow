/**
 * conceptflow-mini/procedural — generators for the scenery of a backdrop.
 *
 * Each generator builds SVG from deterministic noise (a fixed `seed` always
 * gives the same picture) over the box of the depth layer it is drawn in, and
 * takes every colour as a prop: the generators never choose a colour, the
 * video's palette does. Distances scale with the layer box, so the same call
 * fits a landscape and a portrait frame.
 */
import React from 'react';
import {random, useCurrentFrame} from 'remotion';
import {noise2D} from '@remotion/noise';
import {mix, withAlpha} from './film';
import {useLayerBox, type LayerBox} from './scene';

/** Seeded noise in -1..1 along one axis. */
export function noise1(seed: string, x: number): number {
  return noise2D(seed, x, 0);
}

/** Points of a ridge line across `box`, `steps` segments, heights `y ± amp` from fractal noise. */
export function ridgeLine(
  box: Pick<LayerBox, 'left' | 'width'>,
  y: number,
  amp: number,
  seed: string,
  opts: {steps?: number; rough?: number; scale?: number} = {},
): [number, number][] {
  const {steps = 48, rough = 0.5, scale = 3} = opts;
  const pts: [number, number][] = [];
  for (let i = 0; i <= steps; i += 1) {
    const u = i / steps;
    const n = noise1(seed, u * scale) + rough * noise1(seed + 'b', u * scale * 2.7) * 0.5 + rough * 0.25 * noise1(seed + 'c', u * scale * 6.1);
    pts.push([box.left + u * box.width, y - n * amp]);
  }
  return pts;
}

function closedPath(line: [number, number][], bottom: number): string {
  const first = line[0];
  const last = line[line.length - 1];
  return `M ${first[0]} ${bottom} ` + line.map(([x, y]) => `L ${x} ${y}`).join(' ') + ` L ${last[0]} ${bottom} Z`;
}

/** Jagged mountain ridges filling down to the bottom of the layer. */
export function Ridges({y, amp, color, seed = 'ridges', rough = 0.8}: {y: number; amp: number; color: string; seed?: string; rough?: number}) {
  const box = useLayerBox();
  return <path d={closedPath(ridgeLine(box, y, amp, seed, {steps: 64, rough, scale: 4}), box.top + box.height)} fill={color} />;
}

/** Smooth rolling hills filling down to the bottom of the layer. */
export function Hills({y, amp, color, seed = 'hills'}: {y: number; amp: number; color: string; seed?: string}) {
  const box = useLayerBox();
  return <path d={closedPath(ridgeLine(box, y, amp, seed, {steps: 40, rough: 0.15, scale: 1.6}), box.top + box.height)} fill={color} />;
}

/** Desert dunes: long asymmetric swells. */
export function Dunes({y, amp, color, seed = 'dunes'}: {y: number; amp: number; color: string; seed?: string}) {
  const box = useLayerBox();
  const line = ridgeLine(box, y, amp * 0.5, seed, {steps: 60, rough: 0.1, scale: 1.2}).map(([x, h], i, all): [number, number] => {
    const u = i / (all.length - 1);
    return [x, h - Math.abs(Math.sin(u * Math.PI * 3.2 + noise1(seed + 's', u * 2))) * amp * 0.5];
  });
  return <path d={closedPath(line, box.top + box.height)} fill={color} />;
}

/** A tree crown: overlapping blobs whose radii come from noise. */
export function Canopy({cx, cy, r, color, seed = 'canopy', blobs = 9}: {cx: number; cy: number; r: number; color: string; seed?: string; blobs?: number}) {
  return (
    <g>
      {Array.from({length: blobs}, (_, i) => {
        const a = (i / blobs) * Math.PI * 2;
        const rr = r * (0.38 + 0.18 * (noise1(seed, i * 1.7) + 1));
        const dist = r * 0.5 * (0.6 + 0.4 * (noise1(seed + 'd', i * 2.3) + 1) / 2);
        const tone = mix(color, '#000000', 0.12 * (Math.sin(a) + 1) * 0.5);
        return <circle key={i} cx={cx + Math.cos(a) * dist} cy={cy + Math.sin(a) * dist * 0.8} r={rr} fill={tone} />;
      })}
      <circle cx={cx} cy={cy} r={r * 0.55} fill={color} />
    </g>
  );
}

/** Blades of grass along `baseY`; they sway unless `still`. */
export function Grass({baseY, height, color, count = 80, seed = 'grass', still = false}: {baseY: number; height: number; color: string; count?: number; seed?: string; still?: boolean}) {
  const box = useLayerBox();
  const frame = useCurrentFrame();
  return (
    <g fill={color}>
      {Array.from({length: count}, (_, i) => {
        const x = box.left + random(`${seed}-x-${i}`) * box.width;
        const h = height * (0.5 + random(`${seed}-h-${i}`) * 0.7);
        const w = height * 0.045;
        const lean = (random(`${seed}-l-${i}`) - 0.5) * height * 0.5 + (still ? 0 : Math.sin(frame / 22 + i) * height * 0.08);
        return <path key={i} d={`M ${x - w} ${baseY} Q ${x + lean * 0.4} ${baseY - h * 0.5} ${x + lean} ${baseY - h} Q ${x + lean * 0.3} ${baseY - h * 0.45} ${x + w} ${baseY} Z`} />;
      })}
    </g>
  );
}

/** Soft clouds: stacks of ellipses with a lit top. */
export function Clouds({y, spread, color, shade, count = 5, seed = 'clouds'}: {y: number; spread: number; color: string; shade?: string; count?: number; seed?: string}) {
  const box = useLayerBox();
  return (
    <g>
      {Array.from({length: count}, (_, i) => {
        const cx = box.left + ((i + 0.5 + (random(`${seed}-x-${i}`) - 0.5) * 0.6) / count) * box.width;
        const cy = y + (random(`${seed}-y-${i}`) - 0.5) * spread;
        const w = box.width * (0.07 + random(`${seed}-w-${i}`) * 0.08);
        return (
          <g key={i}>
            <ellipse cx={cx} cy={cy + w * 0.12} rx={w} ry={w * 0.32} fill={shade ?? mix(color, '#000000', 0.1)} />
            <ellipse cx={cx - w * 0.3} cy={cy - w * 0.08} rx={w * 0.55} ry={w * 0.3} fill={color} />
            <ellipse cx={cx + w * 0.25} cy={cy - w * 0.14} rx={w * 0.45} ry={w * 0.28} fill={color} />
          </g>
        );
      })}
    </g>
  );
}

/** Stars of varied size; they twinkle unless `still`. */
export function Stars({count = 90, color, seed = 'stars', still = false, minR = 1, maxR = 3.5}: {count?: number; color: string; seed?: string; still?: boolean; minR?: number; maxR?: number}) {
  const box = useLayerBox();
  const frame = useCurrentFrame();
  return (
    <g fill={color}>
      {Array.from({length: count}, (_, i) => {
        const big = random(`${seed}-r-${i}`);
        const r = minR + big * big * (maxR - minR);
        const twinkle = still ? 1 : 0.65 + 0.35 * Math.sin(frame / 9 + i * 1.7);
        return <circle key={i} cx={box.left + random(`${seed}-x-${i}`) * box.width} cy={box.top + random(`${seed}-y-${i}`) * box.height} r={r} opacity={(0.45 + 0.55 * big) * twinkle} />;
      })}
    </g>
  );
}

/** Faceted rocks standing on `baseY`. */
export function Rocks({baseY, count = 5, size, color, seed = 'rocks'}: {baseY: number; count?: number; size: number; color: string; seed?: string}) {
  const box = useLayerBox();
  return (
    <g>
      {Array.from({length: count}, (_, i) => {
        const cx = box.left + random(`${seed}-x-${i}`) * box.width;
        const s = size * (0.5 + random(`${seed}-s-${i}`));
        const pts = Array.from({length: 7}, (_, k) => {
          const a = Math.PI + (k / 6) * Math.PI;
          const r = s * (0.8 + 0.3 * noise1(seed, i * 5 + k));
          return `${cx + Math.cos(a) * r * 1.2},${baseY + Math.sin(a) * r * 0.8}`;
        });
        return <polygon key={i} points={`${cx - s * 1.2},${baseY} ${pts.join(' ')} ${cx + s * 1.2},${baseY}`} fill={mix(color, '#000000', 0.15 * (i % 3) / 2)} />;
      })}
    </g>
  );
}

/** Branching coral fans standing on `baseY`. */
export function Coral({baseY, count = 4, height, color, seed = 'coral'}: {baseY: number; count?: number; height: number; color: string; seed?: string}) {
  const box = useLayerBox();
  const branch = (x: number, y: number, len: number, angle: number, depth: number, key: string): React.ReactNode => {
    if (depth === 0 || len < 4) return null;
    const x2 = x + Math.cos(angle) * len;
    const y2 = y + Math.sin(angle) * len;
    const turn = 0.45 + 0.2 * noise1(seed + key, depth);
    return (
      <g key={key}>
        <line x1={x} y1={y} x2={x2} y2={y2} stroke={color} strokeWidth={Math.max(3, len * 0.18)} strokeLinecap="round" />
        {branch(x2, y2, len * 0.72, angle - turn, depth - 1, key + 'a')}
        {branch(x2, y2, len * 0.72, angle + turn, depth - 1, key + 'b')}
      </g>
    );
  };
  return (
    <g>
      {Array.from({length: count}, (_, i) => branch(box.left + random(`${seed}-x-${i}`) * box.width, baseY, height * 0.3, -Math.PI / 2, 5, `c${i}`))}
    </g>
  );
}

/** Layered waves across the layer at height `y`; they roll unless `still`. */
export function Waves({y, amp, color, layers = 3, seed = 'waves', still = false}: {y: number; amp: number; color: string; layers?: number; seed?: string; still?: boolean}) {
  const box = useLayerBox();
  const frame = useCurrentFrame();
  return (
    <g>
      {Array.from({length: layers}, (_, i) => {
        const drift = still ? 0 : frame / 40 + i * 2;
        const line = Array.from({length: 49}, (_, k): [number, number] => {
          const u = k / 48;
          return [box.left + u * box.width, y + i * amp * 0.7 - noise2D(seed + i, u * 3 + drift, i) * amp];
        });
        return <path key={i} d={closedPath(line, box.top + box.height)} fill={withAlpha(color, 0.35 + 0.2 * i)} />;
      })}
    </g>
  );
}

/** A convex polygon cut out of `cell` by the perpendicular bisector with `other` (keeps the side of `site`). */
function clipBisector(cell: [number, number][], site: [number, number], other: [number, number]): [number, number][] {
  const mx = (site[0] + other[0]) / 2;
  const my = (site[1] + other[1]) / 2;
  const nx = other[0] - site[0];
  const ny = other[1] - site[1];
  const side = (p: [number, number]) => (p[0] - mx) * nx + (p[1] - my) * ny;
  const out: [number, number][] = [];
  for (let i = 0; i < cell.length; i += 1) {
    const a = cell[i];
    const b = cell[(i + 1) % cell.length];
    const sa = side(a);
    const sb = side(b);
    if (sa <= 0) out.push(a);
    if (sa <= 0 !== sb <= 0) {
      const t = sa / (sa - sb);
      out.push([a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t]);
    }
  }
  return out;
}

/** Voronoi cells of `sites` clipped to `box`, one polygon per site. */
export function voronoiCells(sites: [number, number][], box: Pick<LayerBox, 'left' | 'top' | 'width' | 'height'>): [number, number][][] {
  const rect: [number, number][] = [
    [box.left, box.top],
    [box.left + box.width, box.top],
    [box.left + box.width, box.top + box.height],
    [box.left, box.top + box.height],
  ];
  return sites.map((site, i) => {
    let cell = rect;
    for (let j = 0; j < sites.length && cell.length > 0; j += 1) {
      if (j !== i) cell = clipBisector(cell, site, sites[j]);
    }
    return cell;
  });
}

/** Tissue: Voronoi cells with walls in `wall`, tinted between `a` and `b`, each with a nucleus. */
export function Cells({count = 26, a, b, wall, nucleus, seed = 'cells', opacity = 1}: {count?: number; a: string; b: string; wall: string; nucleus: string; seed?: string; opacity?: number}) {
  const box = useLayerBox();
  const sites = Array.from({length: count}, (_, i): [number, number] => [box.left + random(`${seed}-x-${i}`) * box.width, box.top + random(`${seed}-y-${i}`) * box.height]);
  const cells = voronoiCells(sites, box);
  return (
    <g opacity={opacity}>
      {cells.map((cell, i) => (
        <g key={i}>
          <polygon points={cell.map((p) => p.join(',')).join(' ')} fill={mix(a, b, random(`${seed}-t-${i}`))} stroke={wall} strokeWidth={Math.min(box.width, box.height) * 0.004} strokeLinejoin="round" />
          <circle cx={sites[i][0]} cy={sites[i][1]} r={Math.min(box.width, box.height) * 0.012} fill={nucleus} opacity={0.7} />
        </g>
      ))}
    </g>
  );
}

/** Floating particles (dust, bubbles, pollen) that drift upward unless `still`. */
export function Particles({count = 40, color, size = 4, seed = 'particles', still = false, speed = 0.4}: {count?: number; color: string; size?: number; seed?: string; still?: boolean; speed?: number}) {
  const box = useLayerBox();
  const frame = useCurrentFrame();
  return (
    <g fill={color}>
      {Array.from({length: count}, (_, i) => {
        const rise = still ? 0 : frame * speed * (0.5 + random(`${seed}-v-${i}`));
        const y = box.top + ((random(`${seed}-y-${i}`) * box.height - rise) % box.height + box.height) % box.height;
        const x = box.left + random(`${seed}-x-${i}`) * box.width + (still ? 0 : Math.sin(frame / 30 + i) * size * 2);
        return <circle key={i} cx={x} cy={y} r={size * (0.4 + random(`${seed}-r-${i}`))} opacity={0.2 + 0.5 * random(`${seed}-o-${i}`)} />;
      })}
    </g>
  );
}
