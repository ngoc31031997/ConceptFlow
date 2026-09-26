/**
 * conceptflow-mini/lottie — plays a clip from the channel's Lottie catalog
 * (CR-038). The catalog itself (ids, licences, approval) lives in
 * ../../lottie/manifest.json and is curated by the Creator; a generated script
 * only ever picks an id, never writes or edits Lottie JSON.
 *
 *   <LottieClip id="cat.thinking" x={1500} y={620} size={360}
 *               colors={{'#F5A623': PALETTE.accent}} />
 *
 * `x`/`y` are the CENTRE of the clip and `size` its longer edge, the same
 * convention as the LAYOUT constants in a generated script, so a clip can share
 * coordinates with the shapes around it.
 *
 * Playback is driven by Remotion's frame (via @remotion/lottie), not by the wall
 * clock, so parallel renders of the same composition produce identical frames.
 */
import {Lottie, type LottieAnimationData} from '@remotion/lottie';
import React, {useEffect, useState} from 'react';
import {continueRender, delayRender, Sequence, staticFile} from 'remotion';

export type LottieColorMap = Record<string, string>;

export interface LottieClipProps {
  id: string;
  /** Centre of the clip in the 1920x1080 frame. Defaults to the frame centre. */
  x?: number;
  y?: number;
  /** Length of the clip's longer edge, in px. */
  size?: number;
  /** Replace source colours, e.g. {'#F5A623': PALETTE.accent}. Unmapped colours stay. */
  colors?: LottieColorMap;
  /** Overrides the catalog default; the manifest's `loop` is not readable here. */
  loop?: boolean;
  playbackRate?: number;
  /** Frames to wait (relative to the enclosing Sequence) before the clip starts. */
  startFrame?: number;
  /** Mirror horizontally, e.g. a character that should face the other way. */
  flip?: boolean;
  opacity?: number;
  /**
   * A solid border traced around the clip's whole silhouette, so a dark clip
   * stays readable on any background. Independent of the clip's own colours,
   * and merges overlapping parts into one outline (unlike a per-layer stroke).
   */
  outline?: {color: string; width?: number};
}

const cache = new Map<string, Promise<LottieAnimationData>>();

function loadClip(id: string): Promise<LottieAnimationData> {
  let pending = cache.get(id);
  if (!pending) {
    pending = fetch(staticFile(`lottie/${id}.json`)).then((res) => {
      if (!res.ok) {
        throw new Error(
          `conceptflow-mini: LottieClip id '${id}' not found (${res.status}). ` +
            `Only clips in the approved catalog exist — see lottie/manifest.json.`,
        );
      }
      return res.json();
    });
    cache.set(id, pending);
  }
  return pending;
}

function toHex(components: unknown): string | null {
  if (!Array.isArray(components) || components.length < 3) return null;
  const [r, g, b] = components.slice(0, 3).map((v) => Math.max(0, Math.min(255, Math.round(Number(v) * 255))));
  return '#' + [r, g, b].map((v) => v.toString(16).padStart(2, '0')).join('').toUpperCase();
}

function fromHex(hex: string): [number, number, number] | null {
  const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim());
  if (!m) return null;
  const n = parseInt(m[1], 16);
  return [((n >> 16) & 255) / 255, ((n >> 8) & 255) / 255, (n & 255) / 255];
}

/**
 * Returns a copy of `data` with fill/stroke colours remapped. Deterministic and
 * pure: exact (rounded to 8-bit) match on the source colour, alpha preserved.
 */
export function recolorLottie(data: LottieAnimationData, colors: LottieColorMap): LottieAnimationData {
  const table = new Map<string, [number, number, number]>();
  for (const [from, to] of Object.entries(colors)) {
    const rgb = fromHex(to);
    if (rgb) table.set(from.toUpperCase(), rgb);
  }
  if (table.size === 0) return data;

  const swap = (components: unknown): unknown => {
    const hex = toHex(components);
    const target = hex ? table.get(hex) : undefined;
    if (!target || !Array.isArray(components)) return components;
    return [...target, components[3] ?? 1];
  };

  const walk = (node: unknown): unknown => {
    if (Array.isArray(node)) return node.map(walk);
    if (node && typeof node === 'object') {
      const obj = node as Record<string, unknown>;
      const out: Record<string, unknown> = {};
      for (const [key, value] of Object.entries(obj)) out[key] = walk(value);
      if ((obj.ty === 'fl' || obj.ty === 'st') && obj.c && typeof obj.c === 'object') {
        const c = out.c as {k?: unknown};
        if (Array.isArray(c.k) && c.k.length > 0 && typeof c.k[0] === 'object' && c.k[0] !== null && !Array.isArray(c.k[0])) {
          c.k = (c.k as Array<Record<string, unknown>>).map((kf) => ({...kf, s: swap(kf.s)}));
        } else {
          c.k = swap(c.k);
        }
      }
      return out;
    }
    return node;
  };

  return walk(data) as LottieAnimationData;
}

function outlineFilter(outline: NonNullable<LottieClipProps['outline']>): string {
  // Twelve zero-blur shadows ringed around the silhouette draw a border of
  // `width` px; twelve (not eight) keeps the edge smooth on tight curves.
  const w = outline.width ?? 4;
  return Array.from({length: 12}, (_, i) => {
    const angle = (i / 12) * Math.PI * 2;
    const dx = Math.round(Math.cos(angle) * w * 100) / 100;
    const dy = Math.round(Math.sin(angle) * w * 100) / 100;
    return `drop-shadow(${dx}px ${dy}px 0 ${outline.color})`;
  }).join(' ');
}

function Clip({id, x, y, size, colors, loop, playbackRate, flip, opacity, outline}: LottieClipProps) {
  const [data, setData] = useState<LottieAnimationData | null>(null);
  const [handle] = useState(() => delayRender(`Loading Lottie clip ${id}`));

  useEffect(() => {
    let alive = true;
    loadClip(id).then(
      (loaded) => {
        if (!alive) return;
        setData(colors ? recolorLottie(loaded, colors) : loaded);
        continueRender(handle);
      },
      (err) => {
        // Fail the render loudly rather than shipping a video with a hole in it.
        console.error(err);
        throw err;
      },
    );
    return () => {
      alive = false;
    };
    // colors is an object literal at the call site; its content, not identity, matters.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, handle, JSON.stringify(colors ?? null)]);

  if (!data) return null;
  const w = Number(data.w) || 1;
  const h = Number(data.h) || 1;
  const scale = size / Math.max(w, h);
  return (
    <div
      style={{
        position: 'absolute',
        left: x - (w * scale) / 2,
        top: y - (h * scale) / 2,
        width: w * scale,
        height: h * scale,
        opacity,
        transform: flip ? 'scaleX(-1)' : undefined,
        filter: outline ? outlineFilter(outline) : undefined,
      }}
    >
      <Lottie animationData={data} loop={loop ?? true} playbackRate={playbackRate ?? 1} style={{width: '100%', height: '100%'}} />
    </div>
  );
}

export function LottieClip({
  x = 960,
  y = 540,
  size = 320,
  startFrame = 0,
  opacity = 1,
  ...rest
}: LottieClipProps) {
  const clip = <Clip x={x} y={y} size={size} opacity={opacity} {...rest} />;
  // Sequence restarts the clip's frame counter, which is what "start later" means.
  return startFrame > 0 ? (
    <Sequence from={startFrame} layout="none">
      {clip}
    </Sequence>
  ) : (
    clip
  );
}
