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
}

export interface SegmentsProps {
  segments?: Segment[];
}

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
    return {
      startFrame: Math.max(0, Math.round(startFrame)),
      durationInFrames: Math.max(1, Math.round(durationInFrames)),
    };
  });
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
