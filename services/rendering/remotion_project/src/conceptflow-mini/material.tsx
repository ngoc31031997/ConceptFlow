/**
 * conceptflow-mini/material — forms that are lit by the scene's light.
 *
 * A drawing builds its big forms from `<Shape material color>`. From the
 * shape, its base colour and the light of the scene (`useLook()`), `Shape`
 * computes what a drawing used to hand-paint: a gradient from the lit side
 * to the shadow side, a core shadow, a rim of light on the edge facing the
 * source, a specular spot, surface grain, light bleeding through thin edges,
 * and the glow of emissive things. The same figure placed in another scene
 * lights itself differently.
 *
 * Directions are computed in the drawing's own coordinates: the vector from
 * the figure to the light source is un-rotated and un-mirrored by the box the
 * figure is placed in (`FigureFrame`).
 */
import React from 'react';
import {useVideoConfig} from 'remotion';
import {darken, mix, useLook, withAlpha} from './film';
import {useKit, type MaterialName} from './kits';

/** Where a figure sits in the frame; `Figure` provides it to the shapes inside. */
export interface FigureFrameValue {
  x: number;
  y: number;
  rotate: number;
  flip: boolean;
  /** Size of the figure's drawing box (viewBox width and height). */
  vw: number;
  vh: number;
}

const FigureFrame = React.createContext<FigureFrameValue | null>(null);

/** Provides the placement of a figure to the shapes drawn inside it. */
export function FigureFrameProvider({value, children}: {value: FigureFrameValue; children: React.ReactNode}) {
  return <FigureFrame.Provider value={value}>{children}</FigureFrame.Provider>;
}

export type Vec = [number, number];

/**
 * Unit vector, in a drawing's coordinates, from the figure toward the light.
 * A light on the figure itself shines from above.
 */
export function lightDirection(frame: {x: number; y: number; rotate: number; flip: boolean}, source: Vec): Vec {
  let dx = source[0] - frame.x;
  let dy = source[1] - frame.y;
  const a = (-frame.rotate * Math.PI) / 180;
  const rx = dx * Math.cos(a) - dy * Math.sin(a);
  const ry = dx * Math.sin(a) + dy * Math.cos(a);
  dx = frame.flip ? -rx : rx;
  dy = ry;
  const len = Math.hypot(dx, dy);
  return len < 1e-6 ? [0, -1] : [dx / len, dy / len];
}

const DEFAULT_DEPTH = 0.7;
const THIN_MATERIALS: ReadonlySet<MaterialName> = new Set(['skin', 'leaf', 'organic']);
const GLOSSY_MATERIALS: ReadonlySet<MaterialName> = new Set(['glossy', 'metal', 'glass']);
const TEXTURE_FREQUENCY: Record<string, number> = {
  fur: 0.7,
  fabric: 0.9,
  wood: 0.04,
  stone: 0.25,
  organic: 0.18,
  leaf: 0.3,
};
/** Offset of the core shadow and the rim, as a multiple of the kit's widths. */
const CORE_SHADOW_REACH = 6;

export type ShapeGeometry =
  | {d: string}
  | {ellipse: {cx: number; cy: number; rx: number; ry: number}}
  | {rect: {x: number; y: number; w: number; h: number}}
  | {capsule: {x1: number; y1: number; x2: number; y2: number; r: number}};

/** Path of a capsule: a segment with round caps of radius `r`. */
export function capsulePath(c: {x1: number; y1: number; x2: number; y2: number; r: number}): string {
  const len = Math.hypot(c.x2 - c.x1, c.y2 - c.y1) || 1;
  const nx = (-(c.y2 - c.y1) / len) * c.r;
  const ny = ((c.x2 - c.x1) / len) * c.r;
  return (
    `M ${c.x1 + nx} ${c.y1 + ny} L ${c.x2 + nx} ${c.y2 + ny} ` +
    `A ${c.r} ${c.r} 0 0 0 ${c.x2 - nx} ${c.y2 - ny} L ${c.x1 - nx} ${c.y1 - ny} ` +
    `A ${c.r} ${c.r} 0 0 0 ${c.x1 + nx} ${c.y1 + ny} Z`
  );
}

/** The SVG element of a geometry, painted with `props`; rectangles get the kit's corner `radius`. */
function geometryElement(g: ShapeGeometry, props: React.SVGProps<SVGElement>, radius: number): React.ReactElement {
  if ('d' in g) return <path d={g.d} {...(props as React.SVGProps<SVGPathElement>)} />;
  if ('capsule' in g) return <path d={capsulePath(g.capsule)} {...(props as React.SVGProps<SVGPathElement>)} />;
  if ('ellipse' in g) {
    const e = g.ellipse;
    return <ellipse cx={e.cx} cy={e.cy} rx={e.rx} ry={e.ry} {...(props as React.SVGProps<SVGEllipseElement>)} />;
  }
  const r = g.rect;
  return (
    <rect x={r.x} y={r.y} width={r.w} height={r.h} rx={Math.min(r.w, r.h) * radius} {...(props as React.SVGProps<SVGRectElement>)} />
  );
}

/** Props of `Shape`: a geometry, a material and the base colour from the palette. */
export type ShapeProps = ShapeGeometry & {
  material: MaterialName;
  /** Natural base colour of the form; use PALETTE roles. */
  color: string;
  /** Characteristic length of the drawing in its own units; defaults to the drawing box. */
  unit?: number;
  /** 0 = a flat piece (leaf, paper), 1 = a full round volume (fruit, head); sets how deep the shading goes. */
  depth?: number;
  /** 0..1, strength of the halo of an `emissive` form. */
  glow?: number;
  opacity?: number;
};

/** A form lit by the scene's light according to its material. */
export function Shape(props: ShapeProps) {
  const {material, color, unit, opacity = 1, depth = DEFAULT_DEPTH, glow = 1} = props;
  const geometry = props as ShapeGeometry;
  const look = useLook();
  const kit = useKit();
  const {width, height} = useVideoConfig();
  const placed = React.useContext(FigureFrame);
  const id = React.useId().replace(/[^a-zA-Z0-9_-]/g, '');

  const frame = placed ?? {x: width / 2, y: height / 2, rotate: 0, flip: false, vw: width, vh: height};
  const u = unit ?? Math.min(frame.vw, frame.vh);
  const dir = lightDirection(frame, [look.light.x * width, look.light.y * height]);
  const intensity = look.light.intensity;

  const lit = mix(color, look.light.color, 0.18 * intensity);
  const shadow = mix(darken(color, 0.62), look.ambient, 0.22);
  const rim = kit.rimWidth * u;
  const reach = CORE_SHADOW_REACH * rim;
  const toward: Vec = [dir[0] * reach, dir[1] * reach];
  const away: Vec = [-dir[0] * rim, -dir[1] * rim];

  if (material === 'emissive') {
    return (
      <g opacity={opacity}>
        <defs>
          <filter id={`${id}-glow`} x="-60%" y="-60%" width="220%" height="220%">
            <feGaussianBlur stdDeviation={kit.glowBlur * u * 1.2} />
          </filter>
        </defs>
        {geometryElement(geometry, {fill: color, filter: `url(#${id}-glow)`, opacity: kit.bloom * glow}, kit.radius)}
        {geometryElement(geometry, {fill: mix(color, '#FFFFFF', 0.22)}, kit.radius)}
      </g>
    );
  }

  const gradient = {x1: 0.5 + dir[0] * 0.5, y1: 0.5 + dir[1] * 0.5, x2: 0.5 - dir[0] * 0.5, y2: 0.5 - dir[1] * 0.5};
  const specular = GLOSSY_MATERIALS.has(material) ? kit.specular[material as 'glossy' | 'metal' | 'glass'] : 0;
  const grain = kit.texture[material as keyof typeof kit.texture] ?? 0;
  const thin = THIN_MATERIALS.has(material);

  return (
    <g opacity={opacity}>
      <defs>
        <linearGradient id={`${id}-base`} {...gradient}>
          <stop offset="0" stopColor={lit} />
          <stop offset="1" stopColor={mix(lit, shadow, 0.55)} />
        </linearGradient>
        <clipPath id={`${id}-clip`}>{geometryElement(geometry, {}, kit.radius)}</clipPath>
        {/* Core shadow: the form minus itself pushed toward the light leaves a crescent on the far side. */}
        <mask id={`${id}-core`} maskUnits="userSpaceOnUse" x="-100000" y="-100000" width="200000" height="200000">
          {geometryElement(geometry, {fill: '#fff'}, kit.radius)}
          <g transform={`translate(${toward[0]} ${toward[1]})`}>
            {geometryElement(geometry, {fill: '#000'}, kit.radius)}
          </g>
        </mask>
        {/* Rim: the form minus itself pushed away from the light leaves a crescent on the lit edge. */}
        <mask id={`${id}-rim`} maskUnits="userSpaceOnUse" x="-100000" y="-100000" width="200000" height="200000">
          {geometryElement(geometry, {fill: '#fff'}, kit.radius)}
          <g transform={`translate(${away[0]} ${away[1]})`}>
            {geometryElement(geometry, {fill: '#000'}, kit.radius)}
          </g>
        </mask>
        <filter id={`${id}-soft`} x="-20%" y="-20%" width="140%" height="140%">
          <feGaussianBlur stdDeviation={rim * 2.4} />
        </filter>
        {grain > 0 ? (
          <filter id={`${id}-grain`} x="0" y="0" width="100%" height="100%">
            <feTurbulence type="fractalNoise" baseFrequency={(TEXTURE_FREQUENCY[material] ?? 0.3) / Math.max(1, u / 300)} numOctaves="2" seed="3" />
            <feColorMatrix type="matrix" values="0 0 0 0 0  0 0 0 0 0  0 0 0 0 0  0 0 0 1.4 -0.3" />
          </filter>
        ) : null}
      </defs>
      {geometryElement(geometry, {fill: `url(#${id}-base)`}, kit.radius)}
      <g clipPath={`url(#${id}-clip)`}>
        <g mask={`url(#${id}-core)`} filter={`url(#${id}-soft)`}>
          {geometryElement(geometry, {fill: shadow, opacity: kit.shadowStrength * 1.6 * (0.4 + 0.6 * depth)}, kit.radius)}
        </g>
        {grain > 0 ? (
          <rect x={-frame.vw} y={-frame.vh} width={frame.vw * 3} height={frame.vh * 3} filter={`url(#${id}-grain)`} opacity={grain * 0.6} fill="#000" />
        ) : null}
        {thin ? (
          <g filter={`url(#${id}-soft)`} opacity={kit.sss * intensity * 0.6}>
            {geometryElement(geometry, {fill: 'none', stroke: mix(color, look.light.color, 0.6), strokeWidth: rim * 3}, kit.radius)}
          </g>
        ) : null}
      </g>
      <g mask={`url(#${id}-rim)`}>
        {geometryElement(geometry, {fill: look.light.color, opacity: kit.rimStrength * intensity}, kit.radius)}
      </g>
      {specular > 0 ? (
        <g clipPath={`url(#${id}-clip)`}>
          <g transform={`translate(${dir[0] * reach * 0.8} ${dir[1] * reach * 0.8})`} filter={`url(#${id}-soft)`}>
            {geometryElement(geometry, {fill: '#FFFFFF', opacity: specular * intensity * 0.35}, kit.radius)}
          </g>
        </g>
      ) : null}
    </g>
  );
}

/**
 * The shadow an object throws on the ground: stretched away from the light,
 * tinted by the ambient colour. An emissive object throws a light patch
 * instead, with `lit`.
 */
export function GroundShadow({cx, cy, rx, lit = false}: {cx: number; cy: number; rx: number; lit?: boolean}) {
  const look = useLook();
  const kit = useKit();
  const {width, height} = useVideoConfig();
  const placed = React.useContext(FigureFrame);
  const frame = placed ?? {x: width / 2, y: height / 2, rotate: 0, flip: false};
  const dir = lightDirection(frame, [look.light.x * width, look.light.y * height]);
  const stretch = 1 + Math.abs(dir[0]) * 0.8;
  const offset = -dir[0] * rx * 0.35;
  const color = lit ? look.light.color : mix(darken(look.ambient, 0.35), '#000000', 0.4);
  return (
    <ellipse
      cx={cx + offset}
      cy={cy}
      rx={rx * stretch}
      ry={rx * 0.16}
      fill={withAlpha(color, lit ? kit.bloom * 0.5 : kit.shadowStrength * 0.8)}
    />
  );
}
