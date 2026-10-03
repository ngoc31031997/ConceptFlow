/**
 * conceptflow-mini/segments — the Remotion-side half of the two-pass
 * narration contract (feature/remotion-engine).
 *
 * `remotion_renderer.py`'s render() converts each narration_segment's real
 * TTS duration_seconds into a frame range and writes it as
 * `inputProps.segments: {startFrame, durationInFrames}[]` — this module
 * turns that prop into Remotion's own timing primitives so a Creator's
 * script never touches frame math by hand.
 *
 * Contract: segments are expected to be contiguous (each startFrame equals
 * the previous segment's end); a gap renders as empty frames.
 */
import type {CalculateMetadataFunction} from 'remotion';
import {Sequence} from 'remotion';
import React from 'react';

export interface Segment {
  startFrame: number;
  durationInFrames: number;
  /**
   * The frame, from the start of the segment, each of its narration lines
   * starts at (`lines[0]` is 0). Absent when the render has no real line
   * timing (a script written before narration lines, the layout probe).
   */
  lines?: number[];
}

// A type alias, not an interface: only aliases get an implicit index signature,
// which <Composition>'s `Props extends Record<string, unknown>` needs for tsc.
export type SegmentsProps = {
  segments?: Segment[];
};

const FALLBACK_DURATION_IN_FRAMES = 150;

/**
 * Coerces frame values to what <Sequence> accepts: integer start >= 0 and
 * integer duration >= 1 (a very short TTS clip or float rounding upstream
 * would otherwise make Remotion throw mid-render).
 */
function normalizeSegments(segments: Segment[] = []): Segment[] {
  return segments.map((segment, index) => {
    const {startFrame, durationInFrames} = segment;
    if (!Number.isFinite(startFrame) || !Number.isFinite(durationInFrames)) {
      throw new Error(
        `conceptflow-mini: segment ${index} has non-numeric frames: ${JSON.stringify(segment)}`,
      );
    }
    const duration = Math.max(1, Math.round(durationInFrames));
    const normalized: Segment = {
      startFrame: Math.max(0, Math.round(startFrame)),
      durationInFrames: duration,
    };
    if (segment.lines) normalized.lines = normalizeLines(segment.lines, duration);
    return normalized;
  });
}

/**
 * Line starts as whole frames inside the segment, never decreasing, the first
 * at 0 — so `[lines[i], lines[i + 1]]` is always a valid interpolate range.
 */
function normalizeLines(lines: number[], duration: number): number[] {
  let previous = 0;
  return lines.map((start, i) => {
    const frame = i === 0 || !Number.isFinite(start) ? previous : Math.round(start);
    previous = Math.min(duration - 1, Math.max(previous, frame));
    return previous;
  });
}

/**
 * Line starts spread evenly over a shot: stands in for the real timing where
 * none exists (a layout probe, Remotion Studio), so a shot timed by
 * `lines[i]` still plays every change.
 */
export function evenLines(duration: number, count: number): number[] {
  const n = Math.max(1, Math.floor(count));
  return Array.from({length: n}, (_, i) => Math.round((i * duration) / n));
}

/** `[start, end)` frames of line `i` of a shot: from its start to the next line's start, or the shot's end. */
export function lineSpan(lines: number[], i: number, duration: number): [number, number] {
  const start = lines[Math.min(Math.max(0, i), lines.length - 1)] ?? 0;
  const end = i + 1 < lines.length ? lines[i + 1] : duration;
  return [start, Math.max(start + 1, end)];
}

/**
 * Drop-in `calculateMetadata` for a Composition whose duration is entirely
 * decided by its narration segments (the common case — see
 * https://www.remotion.dev/docs/calculate-metadata). Falls back to a 5s
 * placeholder when segments are absent, which only happens if this
 * composition is ever opened outside the render pipeline (e.g. Remotion
 * Studio) with no inputProps supplied.
 */
export const calculateMetadataFromSegments: CalculateMetadataFunction<SegmentsProps> = ({
  props,
}) => {
  const segments = normalizeSegments(props.segments);
  // max, not the last element: out-of-order or overlapping segments must not
  // silently truncate the video.
  const end = Math.max(0, ...segments.map((s) => s.startFrame + s.durationInFrames));
  const durationInFrames = end > 0 ? end : FALLBACK_DURATION_IN_FRAMES;
  return {durationInFrames, props: {...props, segments}};
};

/**
 * Renders `children(index, segment)` once per segment, each wrapped in its
 * own `<Sequence>` so it's only mounted for its narration's real timing
 * window — the Remotion equivalent of Manim's `self.narrate(...)` auto-timed
 * pause, just expressed as a frame range instead of a generator wait.
 */
export function Segments({
  segments,
  children,
}: {
  segments?: Segment[];
  children: (index: number, segment: Segment) => React.ReactNode;
}) {
  return (
    <>
      {normalizeSegments(segments).map((segment, index) => (
        <Sequence
          key={index}
          name={`segment-${index}`}
          from={segment.startFrame}
          durationInFrames={segment.durationInFrames}
        >
          {children(index, segment)}
        </Sequence>
      ))}
    </>
  );
}
