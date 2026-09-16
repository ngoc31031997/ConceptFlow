/**
 * conceptflow-mini/segments — the Remotion-side half of the two-pass
 * narration contract (feature/remotion-engine).
 *
 * `remotion_renderer.py`'s render() converts each narration_segment's real
 * TTS duration_seconds into a frame range and writes it as
 * `inputProps.segments: {startFrame, durationInFrames}[]` — this module
 * turns that prop into Remotion's own timing primitives so a Creator's
 * script never touches frame math by hand.
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
  const segments = props.segments ?? [];
  const last = segments[segments.length - 1];
  const durationInFrames = last ? last.startFrame + last.durationInFrames : 150;
  return {durationInFrames, props};
};

/**
 * Renders `children(index)` once per segment, each wrapped in its own
 * `<Sequence>` so it's only mounted for its narration's real timing window —
 * the Remotion equivalent of Manim's `self.narrate(...)` auto-timed pause,
 * just expressed as a frame range instead of a generator wait.
 */
export function Segments({
  segments,
  children,
}: {
  segments: Segment[];
  children: (index: number) => React.ReactNode;
}) {
  return (
    <>
      {segments.map((segment, index) => (
        <Sequence
          key={index}
          from={segment.startFrame}
          durationInFrames={segment.durationInFrames}
        >
          {children(index)}
        </Sequence>
      ))}
    </>
  );
}
