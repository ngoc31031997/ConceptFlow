/**
 * conceptflow-mini/film — the film look laid over every shot.
 *
 * `Shot` gives a shot its `Look` (light source, ambient colour, atmosphere,
 * colour grade; all colours come from the video's palette) and applies the
 * grade, film grain, a vignette tinted from the ambient colour and, on
 * request, motion blur. Components of the kit that need the scene's light
 * (`Scene`, `Shape`, `GroundShadow`...) read it with `useLook()`; outside a
 * `Shot` they throw instead of guessing a colour.
 */
import React from 'react';
import {AbsoluteFill, useCurrentFrame} from 'remotion';
import {CameraMotionBlur} from '@remotion/motion-blur';
import {useKit, type Look} from './kits';

const LookContext = React.createContext<Look | null>(null);

const GRAIN_TILE = 256;
const MOTION_BLUR_SAMPLES = 6;
const MOTION_BLUR_SHUTTER_ANGLE = 180;
/** Strength of the warm or cool wash at a grade `warmth` of 1. */
const WARMTH_WASH = 0.25;
const WARM = '#FF9A3C';
const COOL = '#3C7BFF';
/** How much of the ambient colour is kept in the vignette's dark corner. */
const VIGNETTE_KEEP = 0.25;

/** Channels of a `#rgb` or `#rrggbb` colour; any other notation is an error. */
export function parseHex(color: string): [number, number, number] {
  const m = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.exec(color.trim());
  if (!m) throw new Error(`Màu "${color}" không phải dạng #rgb hoặc #rrggbb`);
  const h = m[1].length === 3 ? m[1].replace(/./g, (c) => c + c) : m[1];
  return [0, 2, 4].map((i) => parseInt(h.slice(i, i + 2), 16)) as [number, number, number];
}

/** `color` scaled toward black by `keep` (1 = unchanged, 0 = black). */
export function darken(color: string, keep: number): string {
  const [r, g, b] = parseHex(color).map((v) => Math.round(v * keep));
  return `#${[r, g, b].map((v) => v.toString(16).padStart(2, '0')).join('')}`;
}

/** Linear blend of two colours, `t` 0 = `a`, 1 = `b`. */
export function mix(a: string, b: string, t: number): string {
  const ca = parseHex(a);
  const cb = parseHex(b);
  const c = ca.map((v, i) => Math.round(v + (cb[i] - v) * t));
  return `#${c.map((v) => v.toString(16).padStart(2, '0')).join('')}`;
}

/** `#rrggbb` with an alpha channel, `alpha` in 0..1. */
export function withAlpha(color: string, alpha: number): string {
  const [r, g, b] = parseHex(color);
  const a = Math.round(Math.max(0, Math.min(1, alpha)) * 255);
  return `#${[r, g, b, a].map((v) => v.toString(16).padStart(2, '0')).join('')}`;
}

/** The look of the shot being drawn; throws outside a `Shot`. */
export function useLook(): Look {
  const look = React.useContext(LookContext);
  if (!look) throw new Error('Thiếu look của cảnh: bọc shot trong <Shot look={LOOKS.<cảnh>}>');
  return look;
}

/** Provides a look without the film layer, for previews of a single illustration. */
export function LookProvider({look, children}: {look: Look; children: React.ReactNode}) {
  return <LookContext.Provider value={look}>{children}</LookContext.Provider>;
}

function FilmGrain({amount}: {amount: number}) {
  const frame = useCurrentFrame();
  const seed = frame % 8;
  const dx = (frame * 37) % GRAIN_TILE;
  const dy = (frame * 91) % GRAIN_TILE;
  return (
    <AbsoluteFill data-cf-layer="light" style={{pointerEvents: 'none', mixBlendMode: 'overlay', opacity: amount * 4}}>
      <svg width="100%" height="100%" style={{position: 'absolute', inset: 0}}>
        <defs>
          <filter id="cf-grain" x="0" y="0" width="100%" height="100%">
            <feTurbulence type="fractalNoise" baseFrequency="0.9" numOctaves="2" seed={seed} stitchTiles="stitch" />
            <feColorMatrix type="saturate" values="0" />
          </filter>
          <pattern id="cf-grain-tile" width={GRAIN_TILE} height={GRAIN_TILE} patternUnits="userSpaceOnUse" x={dx} y={dy}>
            <rect width={GRAIN_TILE} height={GRAIN_TILE} filter="url(#cf-grain)" />
          </pattern>
        </defs>
        <rect width="100%" height="100%" fill="url(#cf-grain-tile)" />
      </svg>
    </AbsoluteFill>
  );
}

/**
 * Corners that fall toward a dark version of the ambient colour, so the
 * vignette belongs to the scene's light instead of being a fixed black.
 */
function TintedVignette({ambient, strength}: {ambient: string; strength: number}) {
  const dark = darken(ambient, VIGNETTE_KEEP);
  return (
    <AbsoluteFill
      data-cf-layer="light"
      style={{
        pointerEvents: 'none',
        backgroundImage: `radial-gradient(ellipse at center, ${withAlpha(dark, 0)} 55%, ${withAlpha(dark, strength)} 100%)`,
      }}
    />
  );
}

/**
 * One shot with its look: colour grade, warm or cool wash, grain, tinted
 * vignette. `motionBlur` adds camera motion blur for a fast camera move.
 */
export function Shot({look, motionBlur = false, children}: {look: Look; motionBlur?: boolean; children: React.ReactNode}) {
  const kit = useKit();
  const {warmth, contrast, saturation} = look.grade;
  const content = (
    <AbsoluteFill style={{filter: `contrast(${1 + contrast}) saturate(${1 + saturation})`}}>{children}</AbsoluteFill>
  );
  return (
    <LookContext.Provider value={look}>
      <AbsoluteFill>
        {motionBlur ? (
          <CameraMotionBlur samples={MOTION_BLUR_SAMPLES} shutterAngle={MOTION_BLUR_SHUTTER_ANGLE}>
            {content}
          </CameraMotionBlur>
        ) : (
          content
        )}
        {warmth !== 0 ? (
          <AbsoluteFill
            data-cf-layer="light"
            style={{
              pointerEvents: 'none',
              mixBlendMode: 'soft-light',
              backgroundColor: withAlpha(warmth > 0 ? WARM : COOL, Math.abs(warmth) * WARMTH_WASH * 4),
            }}
          />
        ) : null}
        <FilmGrain amount={kit.grain} />
        <TintedVignette ambient={look.ambient} strength={kit.vignette} />
      </AbsoluteFill>
    </LookContext.Provider>
  );
}
